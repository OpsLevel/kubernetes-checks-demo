package agent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/opslevel/kubernetes-checks-demo/config"

	"github.com/go-logr/logr"
	"github.com/rs/zerolog/log"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"

	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/workqueue"

	"k8s.io/klog/v2"
)

type controller struct {
	name           string
	integrationId  string
	payloadCheckId string
	clientset      kubernetes.Interface
	queue          workqueue.RateLimitingInterface
	informer       cache.SharedIndexInformer
}

type controllerEvent struct {
	key          string
	resourceType string
	eventType    string
}

func getKubernetesConfig() (*rest.Config, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		configPath := os.Getenv("KUBECONFIG")
		if configPath == "" {
			configPath = os.Getenv("HOME") + "/.kube/config"
		}
		config2, err2 := clientcmd.BuildConfigFromFlags("", configPath)
		if err2 != nil {
			return nil, err2
		}
		return config2, nil
	}
	return config, nil
}

func createKubernetesClient() kubernetes.Interface {
	config, err := getKubernetesConfig()
	if err != nil {
		log.Fatal().Msgf("Unable to create a kubernetes client: %v", err)
	}

	client, err2 := kubernetes.NewForConfig(config)
	if err2 != nil {
		log.Fatal().Msgf("Unable to create a kubernetes client: %v", err)
	}
	// Supress k8s client-go
	klog.SetLogger(logr.Discard())
	return client
}

func createController(channel <-chan struct{}, client kubernetes.Interface, informer cache.SharedIndexInformer, name string, integrationId string, payloadCheckId string) {
	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
	controller := &controller{
		name:           name,
		integrationId:  integrationId,
		payloadCheckId: payloadCheckId,
		clientset:      client,
		informer:       informer,
		queue:          queue,
	}
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    controller.OnAdd,
		UpdateFunc: controller.OnUpdate,
		DeleteFunc: controller.OnDelete,
	})
	go controller.Run(channel)
}

func createControllers(channel <-chan struct{}, c *config.Config) {
	client := createKubernetesClient()
	factory := informers.NewSharedInformerFactory(client, time.Second*time.Duration(c.Resync))
	if c.Deployments {
		createController(channel, client, factory.Apps().V1().Deployments().Informer(), "Deployment", c.IntegrationId, c.PayloadCheckId)
	}
	if c.StatefulSets {
		createController(channel, client, factory.Apps().V1().StatefulSets().Informer(), "StatefulSets", c.IntegrationId, c.PayloadCheckId)
	}
	if c.DaemonSets {
		createController(channel, client, factory.Apps().V1().DaemonSets().Informer(), "DaemonSets", c.IntegrationId, c.PayloadCheckId)
	}
	if c.Jobs {
		createController(channel, client, factory.Batch().V1().Jobs().Informer(), "Job", c.IntegrationId, c.PayloadCheckId)
	}
	if c.CronJobs {
		createController(channel, client, factory.Batch().V1beta1().CronJobs().Informer(), "CronJob", c.IntegrationId, c.PayloadCheckId)
	}
	if c.Services {
		createController(channel, client, factory.Core().V1().Services().Informer(), "Service", c.IntegrationId, c.PayloadCheckId)
	}
	if c.Ingress {
		createController(channel, client, factory.Networking().V1().Ingresses().Informer(), "Ingress", c.IntegrationId, c.PayloadCheckId)
	}
	if c.Configmaps {
		createController(channel, client, factory.Core().V1().ConfigMaps().Informer(), "Configmap", c.IntegrationId, c.PayloadCheckId)
	}
	if c.Secrets {
		createController(channel, client, factory.Core().V1().Secrets().Informer(), "Secrets", c.IntegrationId, c.PayloadCheckId)
	}
	factory.Start(channel)
}

func (c *controller) Run(stopChannel <-chan struct{}) {
	log.Info().Msgf("Starting '%s' Informer", c.name)
	defer runtime.HandleCrash()
	defer c.queue.ShutDown()

	go c.informer.Run(stopChannel)

	if !cache.WaitForNamedCacheSync(c.name, stopChannel, c.informer.HasSynced) {
		runtime.HandleError(fmt.Errorf("Timed out waiting for caches to sync"))
		return
	}

	log.Info().Msg("Agent synced and ready")

	wait.Until(c.runWorker, time.Second, stopChannel)
}

func (c *controller) OnAdd(obj interface{}) {
	var item controllerEvent
	var err error
	item.key, err = cache.MetaNamespaceKeyFunc(obj)
	item.eventType = "create"
	if err == nil {
		c.queue.Add(item)
	}
}

func (c *controller) OnUpdate(old, new interface{}) {
	var item controllerEvent
	var err error
	item.key, err = cache.MetaNamespaceKeyFunc(old)
	item.eventType = "update"
	if err == nil {
		c.queue.Add(item)
	}
}

func (c *controller) OnDelete(obj interface{}) {
	var item controllerEvent
	var err error
	item.key, err = cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	item.eventType = "delete"
	if err == nil {
		c.queue.Add(item)
	}
}

func (c *controller) runWorker() {
	for c.processNextItem() {
		// continue looping
	}
}

// TODO: should likely be a config item
const maxRetries = 5

func (c *controller) processNextItem() bool {
	item, quit := c.queue.Get()

	if quit {
		return false
	}
	defer c.queue.Done(item)
	err := c.processItem(item.(controllerEvent))
	if err == nil {
		// No error, reset the ratelimit counters
		c.queue.Forget(item)
	} else if c.queue.NumRequeues(item) < maxRetries {
		c.queue.AddRateLimited(item)
	} else {
		// err != nil and too many retries
		c.queue.Forget(item)
		runtime.HandleError(err)
	}

	return true
}

type Payload struct {
	Service string                 `json:"service"`
	Check   string                 `json:"check"`
	Data    map[string]interface{} `json:"data"`
}

func (c *controller) post(payload Payload) error {
	url := fmt.Sprintf("https://app.opslevel.com/integrations/payload/%s", c.integrationId)
	jsonData, jsonErr := json.Marshal(payload)
	if jsonErr != nil {
		return jsonErr
	}
	postBody := bytes.NewBuffer(jsonData)
	_, postErr := http.Post(url, "application/json", postBody)
	if postErr != nil {
		return postErr
	}
	return nil
}

func (c *controller) processItem(event controllerEvent) error {
	obj, _, err := c.informer.GetIndexer().GetByKey(event.key)
	if err != nil {
		log.Warn().Msgf("Error fetching object with key %s from store: %v", event.key, err)
		return err
	}
	switch event.eventType {
	case "create":
		//log.Info().Msgf("Processed '%s' of '%s:%s/%s'", event.eventType, c.name, mObj.GetNamespace(), mObj.GetName())
		break
	case "update":
		mObj := obj.(v1.Object)
		var k8s map[string]interface{}
		jsonData, _ := json.Marshal(obj)
		json.Unmarshal(jsonData, &k8s)
		payload := Payload{
			Service: fmt.Sprintf("k8s:%s-%s", mObj.GetName(), mObj.GetNamespace()),
			Check:   c.payloadCheckId,
			Data:    k8s,
		}
		if err = c.post(payload); err != nil {
			return err
		}
		log.Info().Msgf("Processed '%s' of '%s:%s/%s'", event.eventType, c.name, mObj.GetNamespace(), mObj.GetName())
		break
	case "delete":
		//log.Info().Msgf("Processed '%s' of '%s:%s/%s'", event.eventType, c.name, mObj.GetNamespace(), mObj.GetName())
		break
	}
	return nil
}
