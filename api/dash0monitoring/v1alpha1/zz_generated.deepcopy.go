//go:build !ignore_autogenerated

// SPDX-FileCopyrightText: Copyright 2024 Dash0 Inc.
// SPDX-License-Identifier: Apache-2.0

// Code generated by controller-gen. DO NOT EDIT.

package v1alpha1

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Authorization) DeepCopyInto(out *Authorization) {
	*out = *in
	if in.Token != nil {
		in, out := &in.Token, &out.Token
		*out = new(string)
		**out = **in
	}
	if in.SecretRef != nil {
		in, out := &in.SecretRef, &out.SecretRef
		*out = new(SecretRef)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Authorization.
func (in *Authorization) DeepCopy() *Authorization {
	if in == nil {
		return nil
	}
	out := new(Authorization)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Dash0Configuration) DeepCopyInto(out *Dash0Configuration) {
	*out = *in
	in.Authorization.DeepCopyInto(&out.Authorization)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Dash0Configuration.
func (in *Dash0Configuration) DeepCopy() *Dash0Configuration {
	if in == nil {
		return nil
	}
	out := new(Dash0Configuration)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Dash0Monitoring) DeepCopyInto(out *Dash0Monitoring) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Dash0Monitoring.
func (in *Dash0Monitoring) DeepCopy() *Dash0Monitoring {
	if in == nil {
		return nil
	}
	out := new(Dash0Monitoring)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *Dash0Monitoring) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Dash0MonitoringList) DeepCopyInto(out *Dash0MonitoringList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]Dash0Monitoring, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Dash0MonitoringList.
func (in *Dash0MonitoringList) DeepCopy() *Dash0MonitoringList {
	if in == nil {
		return nil
	}
	out := new(Dash0MonitoringList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *Dash0MonitoringList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Dash0MonitoringSpec) DeepCopyInto(out *Dash0MonitoringSpec) {
	*out = *in
	in.Export.DeepCopyInto(&out.Export)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Dash0MonitoringSpec.
func (in *Dash0MonitoringSpec) DeepCopy() *Dash0MonitoringSpec {
	if in == nil {
		return nil
	}
	out := new(Dash0MonitoringSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Dash0MonitoringStatus) DeepCopyInto(out *Dash0MonitoringStatus) {
	*out = *in
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make([]v1.Condition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Dash0MonitoringStatus.
func (in *Dash0MonitoringStatus) DeepCopy() *Dash0MonitoringStatus {
	if in == nil {
		return nil
	}
	out := new(Dash0MonitoringStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Dash0OperatorConfiguration) DeepCopyInto(out *Dash0OperatorConfiguration) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Dash0OperatorConfiguration.
func (in *Dash0OperatorConfiguration) DeepCopy() *Dash0OperatorConfiguration {
	if in == nil {
		return nil
	}
	out := new(Dash0OperatorConfiguration)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *Dash0OperatorConfiguration) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Dash0OperatorConfigurationList) DeepCopyInto(out *Dash0OperatorConfigurationList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]Dash0OperatorConfiguration, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Dash0OperatorConfigurationList.
func (in *Dash0OperatorConfigurationList) DeepCopy() *Dash0OperatorConfigurationList {
	if in == nil {
		return nil
	}
	out := new(Dash0OperatorConfigurationList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *Dash0OperatorConfigurationList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Dash0OperatorConfigurationSpec) DeepCopyInto(out *Dash0OperatorConfigurationSpec) {
	*out = *in
	if in.Export != nil {
		in, out := &in.Export, &out.Export
		*out = new(Export)
		(*in).DeepCopyInto(*out)
	}
	out.SelfMonitoring = in.SelfMonitoring
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Dash0OperatorConfigurationSpec.
func (in *Dash0OperatorConfigurationSpec) DeepCopy() *Dash0OperatorConfigurationSpec {
	if in == nil {
		return nil
	}
	out := new(Dash0OperatorConfigurationSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Dash0OperatorConfigurationStatus) DeepCopyInto(out *Dash0OperatorConfigurationStatus) {
	*out = *in
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make([]v1.Condition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Dash0OperatorConfigurationStatus.
func (in *Dash0OperatorConfigurationStatus) DeepCopy() *Dash0OperatorConfigurationStatus {
	if in == nil {
		return nil
	}
	out := new(Dash0OperatorConfigurationStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Export) DeepCopyInto(out *Export) {
	*out = *in
	if in.Dash0 != nil {
		in, out := &in.Dash0, &out.Dash0
		*out = new(Dash0Configuration)
		(*in).DeepCopyInto(*out)
	}
	if in.Http != nil {
		in, out := &in.Http, &out.Http
		*out = new(HttpConfiguration)
		(*in).DeepCopyInto(*out)
	}
	if in.Grpc != nil {
		in, out := &in.Grpc, &out.Grpc
		*out = new(GrpcConfiguration)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Export.
func (in *Export) DeepCopy() *Export {
	if in == nil {
		return nil
	}
	out := new(Export)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *GrpcConfiguration) DeepCopyInto(out *GrpcConfiguration) {
	*out = *in
	if in.Headers != nil {
		in, out := &in.Headers, &out.Headers
		*out = make([]Header, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new GrpcConfiguration.
func (in *GrpcConfiguration) DeepCopy() *GrpcConfiguration {
	if in == nil {
		return nil
	}
	out := new(GrpcConfiguration)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Header) DeepCopyInto(out *Header) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Header.
func (in *Header) DeepCopy() *Header {
	if in == nil {
		return nil
	}
	out := new(Header)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *HttpConfiguration) DeepCopyInto(out *HttpConfiguration) {
	*out = *in
	if in.Headers != nil {
		in, out := &in.Headers, &out.Headers
		*out = make([]Header, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new HttpConfiguration.
func (in *HttpConfiguration) DeepCopy() *HttpConfiguration {
	if in == nil {
		return nil
	}
	out := new(HttpConfiguration)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *SecretRef) DeepCopyInto(out *SecretRef) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SecretRef.
func (in *SecretRef) DeepCopy() *SecretRef {
	if in == nil {
		return nil
	}
	out := new(SecretRef)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *SelfMonitoring) DeepCopyInto(out *SelfMonitoring) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SelfMonitoring.
func (in *SelfMonitoring) DeepCopy() *SelfMonitoring {
	if in == nil {
		return nil
	}
	out := new(SelfMonitoring)
	in.DeepCopyInto(out)
	return out
}
