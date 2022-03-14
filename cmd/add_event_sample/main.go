package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
)

func panicError(err error) {
	if err != nil {
		panic(err)
	}
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}

func initClientSet() *kubernetes.Clientset {
	var kubeconfig *string
	if home := homeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, "Go", "config", "red_kubeconfig.yaml"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()
	restConfig, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	panicError(err)
	return kubernetes.NewForConfigOrDie(restConfig)
}

func main() {
	clientSet := initClientSet()
	// 初始化 sharedInformerFactory
	sharedInformerFactory := informers.NewSharedInformerFactory(clientSet, 0)
	// 生成 podInformer
	podInformer := sharedInformerFactory.Core().V1().Pods()
	// 生成具体 informer/indexer
	informer := podInformer.Informer()
	indexer := podInformer.Lister()
	// 添加 Event 事件处理函数
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			podObj := obj.(*corev1.Pod)
			fmt.Printf("Add PodName: %s\n", podObj.GetName())
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			oldPodObj := oldObj.(*corev1.Pod)
			newPodObj := newObj.(*corev1.Pod)

			fmt.Printf("Old PodName: %s\n", oldPodObj.GetName())
			fmt.Printf("New PodName: %s\n", newPodObj.GetName())
		},
		DeleteFunc: func(obj interface{}) {
			podObj := obj.(*corev1.Pod)
			fmt.Printf("Delete PodName: %s\n", podObj.GetName())
		},
	})

	stopCh := make(chan struct{})
	defer close(stopCh)
	// 启动 informer
	sharedInformerFactory.Start(stopCh)
	// 等待同步完成
	sharedInformerFactory.WaitForCacheSync(stopCh)

	// 利用 indexer 获取所有 Pod 资源
	pods, err := indexer.List(labels.Everything())
	panicError(err)
	for _, items := range pods {
		fmt.Printf("namespace: %s, podName:%s\n", items.GetNamespace(), items.GetName())
	}
	<-stopCh
}
