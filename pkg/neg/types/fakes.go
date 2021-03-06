/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package types

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/meta"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/ingress-gce/pkg/composite"
	"k8s.io/ingress-gce/pkg/utils"
)

const (
	TestZone1     = "zone1"
	TestZone2     = "zone2"
	TestInstance1 = "instance1"
	TestInstance2 = "instance2"
	TestInstance3 = "instance3"
	TestInstance4 = "instance4"
	TestInstance5 = "instance5"
	TestInstance6 = "instance6"
)

type fakeZoneGetter struct {
	zoneInstanceMap map[string]sets.String
}

func NewFakeZoneGetter() *fakeZoneGetter {
	return &fakeZoneGetter{
		zoneInstanceMap: map[string]sets.String{
			TestZone1: sets.NewString(TestInstance1, TestInstance2),
			TestZone2: sets.NewString(TestInstance3, TestInstance4, TestInstance5, TestInstance6),
		},
	}
}

func (f *fakeZoneGetter) ListZones() ([]string, error) {
	ret := []string{}
	for key := range f.zoneInstanceMap {
		ret = append(ret, key)
	}
	return ret, nil
}
func (f *fakeZoneGetter) GetZoneForNode(name string) (string, error) {
	for zone, instances := range f.zoneInstanceMap {
		if instances.Has(name) {
			return zone, nil
		}
	}
	return "", NotFoundError
}

type FakeNetworkEndpointGroupCloud struct {
	NetworkEndpointGroups map[string][]*composite.NetworkEndpointGroup
	NetworkEndpoints      map[string][]*composite.NetworkEndpoint
	Subnetwork            string
	Network               string
	mu                    sync.Mutex
}

// DEPRECATED: Please do not use this mock function. Use the pkg/neg/types/mock.go instead.
func NewFakeNetworkEndpointGroupCloud(subnetwork, network string) NetworkEndpointGroupCloud {
	return &FakeNetworkEndpointGroupCloud{
		Subnetwork:            subnetwork,
		Network:               network,
		NetworkEndpointGroups: map[string][]*composite.NetworkEndpointGroup{},
		NetworkEndpoints:      map[string][]*composite.NetworkEndpoint{},
	}
}

var NotFoundError = utils.FakeGoogleAPINotFoundErr()

func (f *FakeNetworkEndpointGroupCloud) GetNetworkEndpointGroup(name string, zone string, version meta.Version) (*composite.NetworkEndpointGroup, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	negs, ok := f.NetworkEndpointGroups[zone]
	if ok {
		for _, neg := range negs {
			if neg.Name == name {
				return neg, nil
			}
		}
	}
	return nil, NotFoundError
}

func networkEndpointKey(name, zone string) string {
	return fmt.Sprintf("%s-%s", zone, name)
}

func (f *FakeNetworkEndpointGroupCloud) ListNetworkEndpointGroup(zone string, version meta.Version) ([]*composite.NetworkEndpointGroup, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.NetworkEndpointGroups[zone], nil
}

func (f *FakeNetworkEndpointGroupCloud) AggregatedListNetworkEndpointGroup(version meta.Version) (map[*meta.Key]*composite.NetworkEndpointGroup, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	result := make(map[*meta.Key]*composite.NetworkEndpointGroup)
	for zone, negs := range f.NetworkEndpointGroups {
		for _, neg := range negs {
			result[&meta.Key{Zone: zone}] = neg
		}
	}
	return result, nil
}

func (f *FakeNetworkEndpointGroupCloud) CreateNetworkEndpointGroup(neg *composite.NetworkEndpointGroup, zone string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	neg.SelfLink = cloud.NewNetworkEndpointGroupsResourceID("mock-project", zone, neg.Name).SelfLink(meta.VersionAlpha)
	if _, ok := f.NetworkEndpointGroups[zone]; !ok {
		f.NetworkEndpointGroups[zone] = []*composite.NetworkEndpointGroup{}
	}
	f.NetworkEndpointGroups[zone] = append(f.NetworkEndpointGroups[zone], neg)
	f.NetworkEndpoints[networkEndpointKey(neg.Name, zone)] = []*composite.NetworkEndpoint{}
	return nil
}

func (f *FakeNetworkEndpointGroupCloud) DeleteNetworkEndpointGroup(name string, zone string, version meta.Version) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.NetworkEndpoints, networkEndpointKey(name, zone))
	negs := f.NetworkEndpointGroups[zone]
	newList := []*composite.NetworkEndpointGroup{}
	found := false
	for _, neg := range negs {
		if neg.Name == name {
			found = true
			continue
		}
		newList = append(newList, neg)
	}
	if !found {
		return NotFoundError
	}
	f.NetworkEndpointGroups[zone] = newList
	return nil
}

func (f *FakeNetworkEndpointGroupCloud) AttachNetworkEndpoints(name, zone string, endpoints []*composite.NetworkEndpoint, version meta.Version) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.NetworkEndpoints[networkEndpointKey(name, zone)] = append(f.NetworkEndpoints[networkEndpointKey(name, zone)], endpoints...)
	return nil
}

func (f *FakeNetworkEndpointGroupCloud) DetachNetworkEndpoints(name, zone string, endpoints []*composite.NetworkEndpoint, version meta.Version) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	newList := []*composite.NetworkEndpoint{}
	for _, ne := range f.NetworkEndpoints[networkEndpointKey(name, zone)] {
		found := false
		for _, remove := range endpoints {
			if reflect.DeepEqual(*ne, *remove) {
				found = true
				break
			}
		}
		if found {
			continue
		}
		newList = append(newList, ne)
	}
	f.NetworkEndpoints[networkEndpointKey(name, zone)] = newList
	return nil
}

func (f *FakeNetworkEndpointGroupCloud) ListNetworkEndpoints(name, zone string, showHealthStatus bool, version meta.Version) ([]*composite.NetworkEndpointWithHealthStatus, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	ret := []*composite.NetworkEndpointWithHealthStatus{}
	nes, ok := f.NetworkEndpoints[networkEndpointKey(name, zone)]
	if !ok {
		return nil, NotFoundError
	}
	for _, ne := range nes {
		ret = append(ret, &composite.NetworkEndpointWithHealthStatus{NetworkEndpoint: ne})
	}
	return ret, nil
}

func (f *FakeNetworkEndpointGroupCloud) NetworkURL() string {
	return f.Network
}

func (f *FakeNetworkEndpointGroupCloud) SubnetworkURL() string {
	return f.Subnetwork
}
