// SPDX-FileCopyrightText: Copyright 2024 Dash0 Inc.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	otelmetric "go.opentelemetry.io/otel/metric"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	oteltrace "go.opentelemetry.io/otel/trace"
	tracenoop "go.opentelemetry.io/otel/trace/noop"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type Settings struct {
	Clientset                  *kubernetes.Clientset
	NodeName                   string
	ConfigMapNamespace         string
	ConfigMapName              string
	FileLogOffsetDirectoryPath string
}

type patch struct {
	BinaryData map[string]string `json:"binaryData,omitempty"`
}

const (
	metricNamePrefix = "dash0operator.filelogoffsetsynch."
)

var (
	currentValue string

	tracerProvider oteltrace.TracerProvider
	tracer         = otel.Tracer("dash0.com/operator/filelogoffsetsynch")

	meterProvider              otelmetric.MeterProvider
	metricNameCompressedSize   = fmt.Sprintf("%s.%s", metricNamePrefix, "update.compressed_size")
	offsetFileSize             otelmetric.Int64Gauge
	metricNameUpdateCounter    = fmt.Sprintf("%s.%s", metricNamePrefix, "update.counter")
	updateCountMeter           otelmetric.Int64Counter
	metricNameUpdateDuration   = fmt.Sprintf("%s.%s", metricNamePrefix, "update.duration")
	updateDurationSecondsMeter otelmetric.Float64Histogram
)

// TODO Add support for sending_queue on separate exporter
// TODO Set up compaction
// TODO Set up metrics & logs
func main() {
	mode := flag.String("mode", "synch",
		"if set to 'init', it will fetch the offset files from the configmap and store it to the "+
			"path stored at ${FILELOG_OFFSET_DIRECTORY_PATH}; synch mode instead will persist the offset "+
			"files at regular intervals")

	flag.Parse()

	if otherArgs := flag.Args(); len(otherArgs) > 0 {
		log.Fatalln("Unexpected arguments: " + strings.Join(otherArgs, ","))
	}

	configMapNamespace, isSet := os.LookupEnv("K8S_CONFIGMAP_NAMESPACE")
	if !isSet {
		log.Fatalln("Required env var 'K8S_CONFIGMAP_NAMESPACE' is not set")
	}

	configMapName, isSet := os.LookupEnv("K8S_CONFIGMAP_NAME")
	if !isSet {
		log.Fatalln("Required env var 'K8S_CONFIGMAP_NAME' is not set")
	}

	podUid, isSet := os.LookupEnv("K8S_POD_UID")
	if !isSet {
		log.Println("Env var 'K8S_POD_UID' is not set")
	}

	nodeName, isSet := os.LookupEnv("K8S_NODE_NAME")
	if !isSet {
		log.Println("Env var 'K8S_NODE_NAME' is not set")
	}

	fileLogOffsetDirectoryPath, isSet := os.LookupEnv("FILELOG_OFFSET_DIRECTORY_PATH")
	if !isSet {
		log.Fatalln("Required env var 'FILELOG_OFFSET_DIRECTORY_PATH' is not set")
	}

	ctx := context.Background()

	// creates the in-cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		log.Fatalf("Cannot create the Kube API client: %v", err)
	}

	var doMeterShutdown func(ctx context.Context) error
	var doTracerShutdown func(ctx context.Context) error

	if _, isSet = os.LookupEnv("OTEL_EXPORTER_OTLP_ENDPOINT"); isSet {
		var metricExporter metric.Exporter
		var spanExporter trace.SpanExporter

		protocol, isProtocolSet := os.LookupEnv("OTEL_EXPORTER_OTLP_PROTOCOL")
		if !isProtocolSet {
			// http/protobuf is the default transport protocol, see spec:
			// https://github.com/open-telemetry/opentelemetry-specification/blob/main/specification/protocol/exporter.md
			protocol = "http/protobuf"
		}

		switch protocol {
		case "grpc":
			if metricExporter, err = otlpmetricgrpc.New(ctx); err != nil {
				log.Fatalf("Cannot create the OTLP gRPC metrics exporter: %v", err)
			}
			if spanExporter, err = otlptracegrpc.New(ctx); err != nil {
				log.Fatalf("Cannot create the OTLP gRPC span exporter: %v", err)
			}
		case "http/protobuf":
			if metricExporter, err = otlpmetrichttp.New(ctx); err != nil {
				log.Fatalf("Cannot create the OTLP HTTP metrics exporter: %v", err)
			}
			if spanExporter, err = otlptracehttp.New(ctx); err != nil {
				log.Fatalf("Cannot create the OTLP HTTP span exporter: %v", err)
			}
		case "http/json":
			log.Fatalf("Cannot create the OTLP HTTP exporter: the protocol 'http/json' is currently unsupported")
		default:
			log.Fatalf("Unexpected OTLP protocol set as value of the 'OTEL_EXPORTER_OTLP_PROTOCOL' environment variable: %v", protocol)
		}

		r, err := resource.New(ctx,
			resource.WithAttributes(
				semconv.K8SPodUID(podUid),
				semconv.K8SNodeName(nodeName),
			),
		)
		if err != nil {
			log.Fatalf("Cannot setup the OTLP resource: %v", err)
		}

		sdkMeterProvider := metric.NewMeterProvider(
			metric.WithResource(r),
			metric.WithReader(metric.NewPeriodicReader(metricExporter, metric.WithTimeout(1*time.Second))),
		)
		sdkTracerProvider := trace.NewTracerProvider(
			trace.WithResource(r),
			trace.WithBatcher(spanExporter, trace.WithBatchTimeout(time.Second)),
		)

		meterProvider = sdkMeterProvider
		doMeterShutdown = sdkMeterProvider.Shutdown
		tracerProvider = sdkTracerProvider
		doTracerShutdown = sdkTracerProvider.Shutdown
	} else {
		meterProvider = metricnoop.MeterProvider{}
		doMeterShutdown = func(ctx context.Context) error { return nil }
		tracerProvider = tracenoop.TracerProvider{}
		doTracerShutdown = func(ctx context.Context) error { return nil }
	}

	otel.SetMeterProvider(meterProvider)
	otel.SetTracerProvider(tracerProvider)

	meter := meterProvider.Meter("dash0.operator.filelog_offset_synch")

	if offsetFileSize, err = meter.Int64Gauge(
		metricNameCompressedSize,
		otelmetric.WithUnit("By"),
		otelmetric.WithDescription("The size of the compressed offset file"),
	); err != nil {
		log.Fatalf("Cannot setup the OTLP meter for the offset file size gauge: %v", err)
	}

	if updateCountMeter, err = meter.Int64Counter(
		metricNameUpdateCounter,
		otelmetric.WithUnit("1"),
		otelmetric.WithDescription("Counter of how many times the synch process for filelog offsets occurs, and how many times it succeeds"),
	); err != nil {
		log.Fatalf("Cannot setup the OTLP meter for the synch counter: %v", err)
	}

	if updateDurationSecondsMeter, err = meter.Float64Histogram(
		metricNameUpdateDuration,
		otelmetric.WithUnit("1s"),
		otelmetric.WithDescription("Counter of how long it takes for the synch process for filelog offsets to complete"),
	); err != nil {
		log.Fatalf("Cannot setup the OTLP meter for the synch duration histogram: %v", err)
	}

	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("Cannot create the Kube API client: %v\n", err)
	}

	settings := &Settings{
		Clientset:                  clientset,
		NodeName:                   nodeName,
		ConfigMapNamespace:         configMapNamespace,
		ConfigMapName:              configMapName,
		FileLogOffsetDirectoryPath: fileLogOffsetDirectoryPath,
	}

	switch *mode {
	case "init":
		if restoredFiles, err := initOffsets(ctx, settings); err != nil {
			log.Fatalf("No offset files restored: %v\n", err)
		} else if restoredFiles == 0 {
			log.Println("No offset files restored")
		}
	case "synch":
		if err := synchOffsets(ctx, settings); err != nil {
			log.Fatalf("An error occurred while synching file offsets to configmap: %v\n", err)
		}
	}

	if meterProvider != nil {
		timeoutCtx, cancelFun := context.WithTimeout(ctx, time.Second)
		if err = doMeterShutdown(timeoutCtx); err != nil {
			log.Printf("Failed to shutdown metrics provider, metrics data nay have been lost:%v\n", err)
		}
		cancelFun()
	}
	if tracerProvider != nil {
		timeoutCtx, cancelFun := context.WithTimeout(ctx, time.Second)
		if err = doTracerShutdown(timeoutCtx); err != nil {
			log.Printf("Failed to shutdown tracer provider, metrics data nay have been lost:%v\n", err)
		}
		cancelFun()
	}
}

func initOffsets(ctx context.Context, settings *Settings) (int, error) {
	configMap, err := settings.Clientset.CoreV1().ConfigMaps(settings.ConfigMapNamespace).Get(ctx, settings.ConfigMapName, metav1.GetOptions{})
	if err != nil {
		return 0, fmt.Errorf("cannot retrieve %v/%v config map: %w", settings.ConfigMapNamespace, settings.ConfigMapName, err)
	}

	offsetBytes, isSet := configMap.BinaryData[settings.NodeName]
	if !isSet {
		// No previous state found
		return 0, nil
	}

	gr, err := gzip.NewReader(bytes.NewReader(offsetBytes))
	if err != nil {
		return 0, fmt.Errorf("cannot uncompress '%v' field of %v/%v config map: %w", settings.NodeName, settings.ConfigMapNamespace, settings.ConfigMapName, err)
	}

	tr := tar.NewReader(gr)

	restoredFiles := 0
	for {
		// Reached the end of the tarfile
		archiveFullyRead, hasRestoredFile, err := restoreFile(tr)

		if err != nil {
			return restoredFiles, err
		}

		if hasRestoredFile {
			restoredFiles += 1
		}

		if archiveFullyRead {
			break
		}
	}

	return restoredFiles, nil
}

type IsArchiveOver bool

type HasRestoredFileFromArchive bool

func restoreFile(tr *tar.Reader) (IsArchiveOver, HasRestoredFileFromArchive, error) {
	nextHeader, err := tr.Next()

	switch {
	case err == io.EOF || nextHeader == nil:
		return true, false, nil
	case err != nil:
		return false, false, fmt.Errorf("cannot read next archive header: %w", err)
	}

	switch nextHeader.Typeflag {
	case tar.TypeDir:
		if _, err := os.Stat(nextHeader.Name); err != nil {
			if err := os.Mkdir(nextHeader.Name, 0755); err != nil {
				return false, false, fmt.Errorf("cannot create directory '%v': %w", nextHeader.Name, err)
			}
			log.Printf("Restored directory '%v'\n", nextHeader.Name)
		}
		return false, false, nil
	case tar.TypeReg:
		file, err := os.OpenFile(nextHeader.Name, os.O_CREATE|os.O_TRUNC|os.O_RDWR, os.FileMode(nextHeader.Mode))
		if err != nil {
			return false, false, fmt.Errorf("cannot create file '%v': %w", nextHeader.Name, err)
		}

		if _, err := io.Copy(file, tr); err != nil {
			return false, false, fmt.Errorf("cannot write %v bytes to file '%v': %w", nextHeader.Size, nextHeader.Name, err)
		}

		if err := file.Close(); err != nil {
			return false, false, fmt.Errorf("cannot close file '%v': %w", nextHeader.Name, err)
		}

		log.Printf("Restored file '%v' (%v bytes)\n", nextHeader.Name, nextHeader.Size)
		return false, true, nil
	default:
		return false, false, fmt.Errorf("unexpected tar type '%v' for entry '%v' (size: %v)", nextHeader.Typeflag, nextHeader.Name, nextHeader.Size)
	}
}

func synchOffsets(ctx context.Context, settings *Settings) error {
	ticker := time.NewTicker(5 * time.Second)
	shutdown := make(chan os.Signal, 1)
	done := make(chan bool, 1)
	signal.Notify(shutdown, syscall.SIGTERM)

	go func() {
		for {
			select {
			case <-ticker.C:
				if err := doSynchOffsetsAndMeasure(ctx, settings); err != nil {
					log.Printf("Cannot update offset files: %v\n", err)
				}
			case <-shutdown:
				ticker.Stop()

				if err := doSynchOffsetsAndMeasure(ctx, settings); err != nil {
					log.Printf("Cannot update offset files on shutdown: %v\n", err)
				}

				done <- true
			}
		}
	}()

	<-done

	return nil
}

type OffsetSizeBytes int
type IsOffsetUpdated bool

func doSynchOffsetsAndMeasure(ctx context.Context, settings *Settings) error {
	ctx, span := tracer.Start(ctx, "synch-offsets")
	defer span.End()

	start := time.Now()

	offsetUpdated, offsetUpdateSize, err := doSynchOffsets(settings)

	elapsed := time.Since(start)
	span.SetAttributes(attribute.Int("elapsed", int(elapsed.Milliseconds())))

	attributes := []attribute.KeyValue{}
	if err != nil {
		span.RecordError(err)
		log.Printf("Cannot update offset files: %v\n", err)

		attributes = append(attributes,
			attribute.String("error.type", "CannotUpdateOffsetFiles"),
			attribute.String("error.message", err.Error()),
		)
	} else if offsetUpdated {
		updateCountMeter.Add(ctx, 1, otelmetric.WithAttributes(
			attributes...,
		))
		offsetFileSize.Record(ctx, int64(offsetUpdateSize))
		updateDurationSecondsMeter.Record(ctx, elapsed.Seconds(), otelmetric.WithAttributes(attributes...))
	}

	return err
}

func doSynchOffsets(settings *Settings) (IsOffsetUpdated, OffsetSizeBytes, error) {
	var buf bytes.Buffer

	// Compress folder to tar, store bytes in configmap
	tarredFiles, err := tarFolder(settings.FileLogOffsetDirectoryPath, &buf)
	if err != nil {
		return false, -1, fmt.Errorf("cannot prepare offset files for storing: %w", err)
	}

	if tarredFiles == 0 {
		return false, -1, nil
	}

	newValue := base64.StdEncoding.EncodeToString(buf.Bytes())

	if newValue == currentValue {
		return false, -1, nil
	}

	if err := patchConfigMap(settings.Clientset, settings.NodeName, settings.ConfigMapNamespace, settings.ConfigMapName, newValue); err != nil {
		return false, -1, fmt.Errorf("cannot store offset files in configmap %v/%v: %w", settings.ConfigMapNamespace, settings.ConfigMapName, err)
	}

	currentValue = newValue
	return false, OffsetSizeBytes(len(buf.Bytes())), nil
}

func patchConfigMap(clientset *kubernetes.Clientset, nodeName string, configMapNamespace string, configMapName string, newValueBase64 string) error {
	patch := &patch{
		BinaryData: map[string]string{
			nodeName: newValueBase64,
		},
	}

	patchBytes, err := json.Marshal(patch)
	if err != nil {
		return fmt.Errorf("cannot marshal configuration map patch: %w", err)
	}

	if _, err := clientset.CoreV1().ConfigMaps(configMapNamespace).Patch(context.Background(), configMapName, types.MergePatchType, patchBytes, metav1.PatchOptions{}); err != nil {
		return fmt.Errorf("cannot update '%v' field of configuration map  %v/%v: %w; merge patch sent: '%v'", nodeName, configMapNamespace, configMapName, err, string(patchBytes))
	}

	return nil
}

func tarFolder(filelogOffsetDirectoryPath string, writer io.Writer) (int, error) {
	gw := gzip.NewWriter(writer)
	tw := tar.NewWriter(gw)

	tarredFiles := 0
	if err := filepath.Walk(filelogOffsetDirectoryPath,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				// An error occurred while walking the path, terminate the walk
				return err
			}

			if hasAddedFileToArchive, err := tarFile(tw, path, info); err != nil {
				return fmt.Errorf("cannot add entry '%v' to archive: %w", path, err)
			} else if hasAddedFileToArchive {
				tarredFiles += 1
			}

			return nil
		}); err != nil {
		return tarredFiles, fmt.Errorf("an error occurred while tar-ing the offset files: %w", err)
	}

	if err := tw.Close(); err != nil {
		return tarredFiles, fmt.Errorf("cannot close the tar writer: %w", err)
	}

	if err := gw.Close(); err != nil {
		return tarredFiles, fmt.Errorf("cannot close the gzip writer: %w", err)
	}

	return tarredFiles, nil
}

type HasAddedFileToArchive bool

func tarFile(writer *tar.Writer, path string, info os.FileInfo) (HasAddedFileToArchive, error) {
	header, err := tar.FileInfoHeader(info, path)
	if err != nil {
		return false, fmt.Errorf("cannot create tar header for file '%v': %w", path, err)
	}

	// Normalize file name in header
	header.Name = filepath.ToSlash(path)

	if err := writer.WriteHeader(header); err != nil {
		return false, err
	}

	if info.IsDir() {
		return false, nil
	}

	file, err := os.Open(path)
	if err != nil {
		return false, err
	}

	if _, err := io.Copy(writer, file); err != nil {
		return false, err
	}

	if err := file.Close(); err != nil {
		return true, err
	}

	return true, nil
}
