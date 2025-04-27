/*
 * This file is part of the KubeVirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright 2018 Red Hat, Inc.
 *
 */

package device_manager

import (
	"context"
	"time"

	pluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"
)

const (
	connectionTimeout = 5 * time.Second
)

type Device interface {
	Start(stop <-chan struct{}) error
	ListAndWatch(*pluginapi.Empty, pluginapi.DevicePlugin_ListAndWatchServer) error
	PreStartContainer(context.Context, *pluginapi.PreStartContainerRequest) (*pluginapi.PreStartContainerResponse, error)
	Allocate(context.Context, *pluginapi.AllocateRequest) (*pluginapi.AllocateResponse, error)
	GetDeviceName() string
	GetInitialized() bool
}
