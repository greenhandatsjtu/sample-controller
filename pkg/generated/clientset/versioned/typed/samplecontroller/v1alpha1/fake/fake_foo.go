/*
Copyright The Kubernetes Authors.

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

// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	"context"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
	v1alpha1 "k8s.io/sample-controller/pkg/apis/samplecontroller/v1alpha1"
)

// FakeFoos implements FooInterface
type FakeFoos struct {
	Fake *FakeSamplecontrollerV1alpha1
	ns   string
}

var foosResource = v1alpha1.SchemeGroupVersion.WithResource("foos")

var foosKind = v1alpha1.SchemeGroupVersion.WithKind("Foo")

// Get takes name of the foo, and returns the corresponding foo object, and an error if there is any.
func (c *FakeFoos) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.Foo, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(foosResource, c.ns, name), &v1alpha1.Foo{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Foo), err
}

// List takes label and field selectors, and returns the list of Foos that match those selectors.
func (c *FakeFoos) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.FooList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(foosResource, foosKind, c.ns, opts), &v1alpha1.FooList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.FooList{ListMeta: obj.(*v1alpha1.FooList).ListMeta}
	for _, item := range obj.(*v1alpha1.FooList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested foos.
func (c *FakeFoos) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(foosResource, c.ns, opts))

}

// Create takes the representation of a foo and creates it.  Returns the server's representation of the foo, and an error, if there is any.
func (c *FakeFoos) Create(ctx context.Context, foo *v1alpha1.Foo, opts v1.CreateOptions) (result *v1alpha1.Foo, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(foosResource, c.ns, foo), &v1alpha1.Foo{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Foo), err
}

// Update takes the representation of a foo and updates it. Returns the server's representation of the foo, and an error, if there is any.
func (c *FakeFoos) Update(ctx context.Context, foo *v1alpha1.Foo, opts v1.UpdateOptions) (result *v1alpha1.Foo, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(foosResource, c.ns, foo), &v1alpha1.Foo{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Foo), err
}

// Delete takes name of the foo and deletes it. Returns an error if one occurs.
func (c *FakeFoos) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteActionWithOptions(foosResource, c.ns, name, opts), &v1alpha1.Foo{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeFoos) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(foosResource, c.ns, listOpts)

	_, err := c.Fake.Invokes(action, &v1alpha1.FooList{})
	return err
}

// Patch applies the patch and returns the patched foo.
func (c *FakeFoos) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.Foo, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(foosResource, c.ns, name, pt, data, subresources...), &v1alpha1.Foo{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Foo), err
}
