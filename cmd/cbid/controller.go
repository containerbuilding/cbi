/*
Copyright The CBI Authors.
Copyright 2017 The Kubernetes Authors.
https://github.com/kubernetes/sample-controller/tree/4d47428cc1926e6cc47f4a5cf4441077ca1b605f

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
	"encoding/json"
	"fmt"
	"time"

	"github.com/golang/glog"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"

	"github.com/containerbuilding/cbi/cmd/cbid/pluginselector"
	cbiv1alpha1 "github.com/containerbuilding/cbi/pkg/apis/cbi/v1alpha1"
	clientset "github.com/containerbuilding/cbi/pkg/client/clientset/versioned"
	cbischeme "github.com/containerbuilding/cbi/pkg/client/clientset/versioned/scheme"
	informers "github.com/containerbuilding/cbi/pkg/client/informers/externalversions"
	listers "github.com/containerbuilding/cbi/pkg/client/listers/cbi/v1alpha1"
	pluginapi "github.com/containerbuilding/cbi/pkg/plugin/api"
)

const controllerAgentName = "cbid"

const (
	// SuccessSynced is used as part of the Event 'reason' when a BuildJob is synced
	SuccessSynced = "Synced"
	// ErrResourceExists is used as part of the Event 'reason' when a BuildJob fails
	// to sync due to a Deployment of the same name already existing.
	ErrResourceExists = "ErrResourceExists"

	// MessageResourceExists is the message used for Events when a resource
	// fails to sync due to a Job already existing
	MessageResourceExists = "Resource %q already exists and is not managed by BuildJob"
	// MessageResourceSynced is the message used for an Event fired when a BuildJob
	// is synced successfully
	MessageResourceSynced = "BuildJob synced successfully"
)

// Controller is the controller implementation for BuildJob resources
type Controller struct {
	// kubeclientset is a standard kubernetes clientset
	kubeclientset kubernetes.Interface
	// cbiclientset is a clientset for our own API group
	cbiclientset    clientset.Interface
	buildJobsLister listers.BuildJobLister

	// workqueue is a rate limited work queue. This is used to queue work to be
	// processed instead of performing it as soon as a change happens. This
	// means we can ensure we only process a fixed amount of resources at a
	// time, and makes it easy to ensure we are never processing the same item
	// simultaneously in two different workers.
	workqueue workqueue.RateLimitingInterface
	// recorder is an event recorder for recording Event resources to the
	// Kubernetes API.
	recorder record.EventRecorder

	pluginselector *pluginselector.PluginSelector
}

// NewController returns a new CBI controller
func NewController(
	kubeclientset kubernetes.Interface,
	cbiclientset clientset.Interface,
	kubeInformerFactory kubeinformers.SharedInformerFactory,
	cbiInformerFactory informers.SharedInformerFactory,
	ps *pluginselector.PluginSelector) *Controller {

	buildJobInformer := cbiInformerFactory.Cbi().V1alpha1().BuildJobs()

	// Create event broadcaster
	// Add CBI types to the default Kubernetes Scheme so Events can be
	// logged for CBI types.
	cbischeme.AddToScheme(scheme.Scheme)
	glog.V(4).Info("Creating event broadcaster")
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(glog.Infof)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeclientset.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: controllerAgentName})

	controller := &Controller{
		kubeclientset:   kubeclientset,
		cbiclientset:    cbiclientset,
		buildJobsLister: buildJobInformer.Lister(),
		workqueue:       workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "BuildJobs"),
		recorder:        recorder,
		pluginselector:  ps,
	}

	glog.Info("Setting up event handlers")
	// Set up an event handler for when BuildJob resources change
	buildJobInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.enqueueBuildJob,
		UpdateFunc: func(old, new interface{}) {
			controller.enqueueBuildJob(new)
		},
	})

	return controller
}

// Run will set up the event handlers for types we are interested in, as well
// as syncing informer caches and starting workers. It will block until stopCh
// is closed, at which point it will shutdown the workqueue and wait for
// workers to finish processing their current work items.
func (c *Controller) Run(threadiness int, stopCh <-chan struct{}) error {
	defer runtime.HandleCrash()
	defer c.workqueue.ShutDown()

	// Start the informer factories to begin populating the informer caches
	glog.Info("Starting BuildJob controller")

	glog.Info("Starting workers")
	// Launch two workers to process BuildJob resources
	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	glog.Info("Started workers")
	<-stopCh
	glog.Info("Shutting down workers")

	return nil
}

// runWorker is a long-running function that will continually call the
// processNextWorkItem function in order to read and process a message on the
// workqueue.
func (c *Controller) runWorker() {
	for c.processNextWorkItem() {
	}
}

// processNextWorkItem will read a single work item off the workqueue and
// attempt to process it, by calling the syncHandler.
func (c *Controller) processNextWorkItem() bool {
	obj, shutdown := c.workqueue.Get()

	if shutdown {
		return false
	}

	// We wrap this block in a func so we can defer c.workqueue.Done.
	err := func(obj interface{}) error {
		// We call Done here so the workqueue knows we have finished
		// processing this item. We also must remember to call Forget if we
		// do not want this work item being re-queued. For example, we do
		// not call Forget if a transient error occurs, instead the item is
		// put back on the workqueue and attempted again after a back-off
		// period.
		defer c.workqueue.Done(obj)
		var key string
		var ok bool
		// We expect strings to come off the workqueue. These are of the
		// form namespace/name. We do this as the delayed nature of the
		// workqueue means the items in the informer cache may actually be
		// more up to date that when the item was initially put onto the
		// workqueue.
		if key, ok = obj.(string); !ok {
			// As the item in the workqueue is actually invalid, we call
			// Forget here else we'd go into a loop of attempting to
			// process a work item that is invalid.
			c.workqueue.Forget(obj)
			runtime.HandleError(fmt.Errorf("expected string in workqueue but got %#v", obj))
			return nil
		}
		// Run the syncHandler, passing it the namespace/name string of the
		// BuildJob resource to be synced.
		if err := c.syncHandler(key); err != nil {
			return fmt.Errorf("error syncing '%s': %s", key, err.Error())
		}
		// Finally, if no error occurs we Forget this item so it does not
		// get queued again until another change happens.
		c.workqueue.Forget(obj)
		glog.Infof("Successfully synced '%s'", key)
		return nil
	}(obj)

	if err != nil {
		runtime.HandleError(err)
		return true
	}

	return true
}

// syncHandler compares the actual state with the desired, and attempts to
// converge the two. It then updates the Status block of the BuildJob resource
// with the current status of the resource.
func (c *Controller) syncHandler(key string) error {
	// Convert the namespace/name string into a distinct namespace and name
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		runtime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}

	// Get the BuildJob resource with this namespace/name
	buildJob, err := c.buildJobsLister.BuildJobs(namespace).Get(name)
	if err != nil {
		// The BuildJob resource may no longer exist, in which case we stop
		// processing.
		if errors.IsNotFound(err) {
			runtime.HandleError(fmt.Errorf("BuildJob '%s' in work queue no longer exists", key))
			return nil
		}

		return err
	}

	if buildJob.Name == "" {
		// We choose to absorb the error here as the worker would requeue the
		// resource otherwise. Instead, the next time the resource is updated
		// the resource will be queued again.
		runtime.HandleError(fmt.Errorf("%s: invalid BuildJob spec", key))
		return nil
	}

	// Resolve CBI plugin client
	pluginClient := c.pluginselector.Select(*buildJob)
	if pluginClient == nil {
		runtime.HandleError(fmt.Errorf("%s: no plugin support this spec", key))
		return nil
	}
	buildJobJSON, err := json.Marshal(buildJob)
	if err != nil {
		return err
	}
	pluginRes, err := pluginClient.Build(context.TODO(), &pluginapi.BuildRequest{
		BuildJobJson: string(buildJobJSON),
	})
	if err != nil {
		return err
	}
	// Finally, we update the status block of the BuildJob resource to reflect the
	// current state of the world
	err = c.updateBuildJobStatus(buildJob, pluginRes.JobName)
	if err != nil {
		return err
	}

	c.recorder.Event(buildJob, corev1.EventTypeNormal, SuccessSynced, MessageResourceSynced)
	return nil
}

func (c *Controller) updateBuildJobStatus(buildJob *cbiv1alpha1.BuildJob, jobName string) error {
	// NEVER modify objects from the store. It's a read-only, local cache.
	// You can use DeepCopy() to make a deep copy of original object and modify this copy
	// Or create a copy manually for better performance
	buildJobCopy := buildJob.DeepCopy()
	buildJobCopy.Status.Job = jobName
	// Until #38113 is merged, we must use Update instead of UpdateStatus to
	// update the Status block of the BuildJob resource. UpdateStatus will not
	// allow changes to the Spec of the resource, which is ideal for ensuring
	// nothing other than resource status has been updated.
	_, err := c.cbiclientset.CbiV1alpha1().BuildJobs(buildJob.Namespace).Update(buildJobCopy)
	return err
}

// enqueueBuildJob takes a BuildJob resource and converts it into a namespace/name
// string which is then put onto the work queue. This method should *not* be
// passed resources of any type other than BuildJob.
func (c *Controller) enqueueBuildJob(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		runtime.HandleError(err)
		return
	}
	c.workqueue.AddRateLimited(key)
}
