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

package main

import (
	"context"
	"fmt"
	batchv1 "k8s.io/api/batch/v1"
	"reflect"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/diff"
	kubeinformers "k8s.io/client-go/informers"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	core "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2/ktesting"

	samplecontroller "k8s.io/sample-controller/pkg/apis/samplecontroller/v1alpha1"
	"k8s.io/sample-controller/pkg/generated/clientset/versioned/fake"
	informers "k8s.io/sample-controller/pkg/generated/informers/externalversions"
)

var (
	alwaysReady        = func() bool { return true }
	noResyncPeriodFunc = func() time.Duration { return 0 }
)

type fixture struct {
	t *testing.T

	client     *fake.Clientset
	kubeclient *k8sfake.Clientset
	// Objects to put in the store.
	fooLister []*samplecontroller.Foo
	jobLister []*batchv1.Job
	// Actions expected to happen on the client.
	kubeactions []core.Action
	actions     []core.Action
	// Objects from here preloaded into NewSimpleFake.
	kubeobjects []runtime.Object
	objects     []runtime.Object
}

func newFixture(t *testing.T) *fixture {
	f := &fixture{}
	f.t = t
	f.objects = []runtime.Object{}
	f.kubeobjects = []runtime.Object{}
	return f
}

func newFoo(name string) *samplecontroller.Foo {
	return &samplecontroller.Foo{
		TypeMeta: metav1.TypeMeta{APIVersion: samplecontroller.SchemeGroupVersion.String()},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: metav1.NamespaceDefault,
		},
		Spec: samplecontroller.FooSpec{
			JobName: fmt.Sprintf("%s-job", name),
			Command: []string{"echo", "hello world"},
		},
	}
}

func (f *fixture) newController(ctx context.Context) (*Controller, informers.SharedInformerFactory, kubeinformers.SharedInformerFactory) {
	f.client = fake.NewSimpleClientset(f.objects...)
	f.kubeclient = k8sfake.NewSimpleClientset(f.kubeobjects...)

	i := informers.NewSharedInformerFactory(f.client, noResyncPeriodFunc())
	k8sI := kubeinformers.NewSharedInformerFactory(f.kubeclient, noResyncPeriodFunc())

	c := NewController(ctx, f.kubeclient, f.client,
		k8sI.Batch().V1().Jobs(), i.Samplecontroller().V1alpha1().Foos())

	c.foosSynced = alwaysReady
	c.jobSynced = alwaysReady
	c.recorder = &record.FakeRecorder{}

	for _, f := range f.fooLister {
		i.Samplecontroller().V1alpha1().Foos().Informer().GetIndexer().Add(f)
	}

	for _, d := range f.jobLister {
		k8sI.Batch().V1().Jobs().Informer().GetIndexer().Add(d)
	}

	return c, i, k8sI
}

func (f *fixture) run(ctx context.Context, fooName string) {
	f.runController(ctx, fooName, true, false)
}

func (f *fixture) runExpectError(ctx context.Context, fooName string) {
	f.runController(ctx, fooName, true, true)
}

func (f *fixture) runController(ctx context.Context, fooName string, startInformers bool, expectError bool) {
	c, i, k8sI := f.newController(ctx)
	if startInformers {
		i.Start(ctx.Done())
		k8sI.Start(ctx.Done())
	}

	err := c.syncHandler(ctx, fooName)
	if !expectError && err != nil {
		f.t.Errorf("error syncing foo: %v", err)
	} else if expectError && err == nil {
		f.t.Error("expected error syncing foo, got nil")
	}

	actions := filterInformerActions(f.client.Actions())
	for i, action := range actions {
		if len(f.actions) < i+1 {
			f.t.Errorf("%d unexpected actions: %+v", len(actions)-len(f.actions), actions[i:])
			break
		}

		expectedAction := f.actions[i]
		checkAction(expectedAction, action, f.t)
	}

	if len(f.actions) > len(actions) {
		f.t.Errorf("%d additional expected actions:%+v", len(f.actions)-len(actions), f.actions[len(actions):])
	}

	k8sActions := filterInformerActions(f.kubeclient.Actions())
	for i, action := range k8sActions {
		if len(f.kubeactions) < i+1 {
			f.t.Errorf("%d unexpected actions: %+v", len(k8sActions)-len(f.kubeactions), k8sActions[i:])
			break
		}

		expectedAction := f.kubeactions[i]
		checkAction(expectedAction, action, f.t)
	}

	if len(f.kubeactions) > len(k8sActions) {
		f.t.Errorf("%d additional expected actions:%+v", len(f.kubeactions)-len(k8sActions), f.kubeactions[len(k8sActions):])
	}
}

// checkAction verifies that expected and actual actions are equal and both have
// same attached resources
func checkAction(expected, actual core.Action, t *testing.T) {
	if !(expected.Matches(actual.GetVerb(), actual.GetResource().Resource) && actual.GetSubresource() == expected.GetSubresource()) {
		t.Errorf("Expected\n\t%#v\ngot\n\t%#v", expected, actual)
		return
	}

	if reflect.TypeOf(actual) != reflect.TypeOf(expected) {
		t.Errorf("Action has wrong type. Expected: %t. Got: %t", expected, actual)
		return
	}

	switch a := actual.(type) {
	case core.CreateActionImpl:
		e, _ := expected.(core.CreateActionImpl)
		expObject := e.GetObject()
		object := a.GetObject()

		if !reflect.DeepEqual(expObject, object) {
			t.Errorf("Action %s %s has wrong object\nDiff:\n %s",
				a.GetVerb(), a.GetResource().Resource, diff.ObjectGoPrintSideBySide(expObject, object))
		}
	case core.UpdateActionImpl:
		e, _ := expected.(core.UpdateActionImpl)
		expObject := e.GetObject()
		object := a.GetObject()

		if !reflect.DeepEqual(expObject, object) {
			t.Errorf("Action %s %s has wrong object\nDiff:\n %s",
				a.GetVerb(), a.GetResource().Resource, diff.ObjectGoPrintSideBySide(expObject, object))
		}
	case core.PatchActionImpl:
		e, _ := expected.(core.PatchActionImpl)
		expPatch := e.GetPatch()
		patch := a.GetPatch()

		if !reflect.DeepEqual(expPatch, patch) {
			t.Errorf("Action %s %s has wrong patch\nDiff:\n %s",
				a.GetVerb(), a.GetResource().Resource, diff.ObjectGoPrintSideBySide(expPatch, patch))
		}
	default:
		t.Errorf("Uncaptured Action %s %s, you should explicitly add a case to capture it",
			actual.GetVerb(), actual.GetResource().Resource)
	}
}

// filterInformerActions filters list and watch actions for testing resources.
// Since list and watch don't change resource state we can filter it to lower
// nose level in our tests.
func filterInformerActions(actions []core.Action) []core.Action {
	ret := []core.Action{}
	for _, action := range actions {
		if len(action.GetNamespace()) == 0 &&
			(action.Matches("list", "foos") ||
				action.Matches("watch", "foos") ||
				action.Matches("list", "jobs") ||
				action.Matches("watch", "jobs")) {
			continue
		}
		ret = append(ret, action)
	}

	return ret
}

func (f *fixture) expectCreateJobAction(d *batchv1.Job) {
	f.kubeactions = append(f.kubeactions, core.NewCreateAction(schema.GroupVersionResource{Resource: "jobs"}, d.Namespace, d))
}

func (f *fixture) expectUpdateJobAction(d *batchv1.Job) {
	f.kubeactions = append(f.kubeactions, core.NewUpdateAction(schema.GroupVersionResource{Resource: "jobs"}, d.Namespace, d))
}

func getKey(foo *samplecontroller.Foo, t *testing.T) string {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(foo)
	if err != nil {
		t.Errorf("Unexpected error getting key for foo %v: %v", foo.Name, err)
		return ""
	}
	return key
}

func TestCreatesJob(t *testing.T) {
	f := newFixture(t)
	foo := newFoo("test")
	_, ctx := ktesting.NewTestContext(t)

	f.fooLister = append(f.fooLister, foo)
	f.objects = append(f.objects, foo)

	expJob := newJob(foo)
	f.expectCreateJobAction(expJob)

	f.run(ctx, getKey(foo, t))
}

func TestDoNothing(t *testing.T) {
	f := newFixture(t)
	foo := newFoo("test")
	_, ctx := ktesting.NewTestContext(t)

	d := newJob(foo)

	f.fooLister = append(f.fooLister, foo)
	f.objects = append(f.objects, foo)
	f.jobLister = append(f.jobLister, d)
	f.kubeobjects = append(f.kubeobjects, d)

	f.run(ctx, getKey(foo, t))
}

func TestNotControlledByUs(t *testing.T) {
	f := newFixture(t)
	foo := newFoo("test")
	_, ctx := ktesting.NewTestContext(t)

	d := newJob(foo)

	d.ObjectMeta.OwnerReferences = []metav1.OwnerReference{}

	f.fooLister = append(f.fooLister, foo)
	f.objects = append(f.objects, foo)
	f.jobLister = append(f.jobLister, d)
	f.kubeobjects = append(f.kubeobjects, d)

	f.runExpectError(ctx, getKey(foo, t))
}

func int32Ptr(i int32) *int32 { return &i }
