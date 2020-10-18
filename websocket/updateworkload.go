package websocket

import (
	"encoding/json"
	"fmt"
	"k8s-ca-websocket/cautils"
	"k8s-ca-websocket/k8sworkloads"
	"time"

	"github.com/golang/glog"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	// corev1beta1 "k8s.io/api/core/v1beta1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func updateWorkload(wlid string, command string) error {
	var err error
	namespace := cautils.GetNamespaceFromWlid(wlid)
	kind := cautils.GetKindFromWlid(wlid)
	clientset, e := kubernetes.NewForConfig(k8sworkloads.GetK8sConfig())
	if e != nil {
		return e
	}
	workload, err := getWorkload(wlid)
	if err != nil {
		return err
	}

	switch kind {
	case "Namespace":
		w := workload.(*corev1.Namespace)
		injectNS(&w.ObjectMeta, command)
		_, err = clientset.CoreV1().Namespaces().Update(w)

	case "Deployment":
		w := workload.(*appsv1.Deployment)
		workloadUpdate(&w.ObjectMeta, command, wlid)
		inject(&w.Spec.Template, command, wlid)
		_, err = clientset.AppsV1().Deployments(namespace).Update(w)

	case "ReplicaSet":
		w := workload.(*appsv1.ReplicaSet)
		workloadUpdate(&w.ObjectMeta, command, wlid)
		inject(&w.Spec.Template, command, wlid)
		_, err = clientset.AppsV1().ReplicaSets(namespace).Update(w)

	case "DaemonSet":
		w := workload.(*appsv1.DaemonSet)
		workloadUpdate(&w.ObjectMeta, command, wlid)
		inject(&w.Spec.Template, command, wlid)
		_, err = clientset.AppsV1().DaemonSets(namespace).Update(w)

	case "StatefulSet":
		w := workload.(*appsv1.StatefulSet)
		workloadUpdate(&w.ObjectMeta, command, wlid)
		inject(&w.Spec.Template, command, wlid)
		w, err = clientset.AppsV1().StatefulSets(namespace).Update(w)

	case "PodTemplate":
		w := workload.(*corev1.PodTemplate)
		workloadUpdate(&w.ObjectMeta, command, wlid)
		inject(&w.Template, command, wlid)
		_, err = clientset.CoreV1().PodTemplates(namespace).Update(w)
	case "CronJob":
		w := workload.(*v1beta1.CronJob)
		workloadUpdate(&w.ObjectMeta, command, wlid)
		inject(&w.Spec.JobTemplate.Spec.Template, command, wlid)
		_, err = clientset.BatchV1beta1().CronJobs(namespace).Update(w)

	case "Job":
		err = fmt.Errorf("")
		// Do nothing
		// w := workload.(*batchv1.Job)
		// inject(&w.Spec.Template, command, wlid)
		// cleanSelector(w.Spec.Selector)
		// err = clientset.BatchV1().Jobs(namespace).Delete(w.Name, &v1.DeleteOptions{})
		// if err == nil {
		// 	w.Status = batchv1.JobStatus{}
		// 	w.ObjectMeta.ResourceVersion = ""
		// 	for {
		// 		_, err = clientset.BatchV1().Jobs(namespace).Get(w.Name, v1.GetOptions{})
		// 		if err != nil {
		// 			break
		// 		}
		// 		time.Sleep(time.Second * 1)
		// 	}
		// 	w, err = clientset.BatchV1().Jobs(namespace).Create(w)
		// }

	case "Pod":
		w := workload.(*corev1.Pod)
		injectPod(&w.ObjectMeta, &w.Spec, command, wlid)
		err = clientset.CoreV1().Pods(namespace).Delete(w.Name, &v1.DeleteOptions{})
		if err == nil {
			w.Status = corev1.PodStatus{}
			w.ObjectMeta.ResourceVersion = ""
			for {
				_, err = clientset.CoreV1().Pods(namespace).Get(w.Name, v1.GetOptions{})
				if err != nil {
					break
				}
				time.Sleep(time.Second * 1)
			}
			_, err = clientset.CoreV1().Pods(namespace).Create(w)
		}
	default:
		err = fmt.Errorf("command %s not supported with kind: %s", command, cautils.GetKindFromWlid(wlid))
	}
	return err

}

func inject(template *corev1.PodTemplateSpec, command, wlid string) {
	switch command {
	case UPDATE:
		injectWlid(&template.ObjectMeta.Annotations, wlid)
		injectTime(&template.ObjectMeta.Annotations)
		injectLabel(&template.ObjectMeta.Labels)

	case SIGN:
		updateLabel(&template.ObjectMeta.Labels)
		injectTime(&template.ObjectMeta.Annotations)
	case REMOVE:
		restoreConatinerCommand(&template.Spec)
		removeCASpec(&template.Spec)
		removeCAMetadata(&template.ObjectMeta)
	}
	removeIDLabels(template.ObjectMeta.Labels)
}

func workloadUpdate(objectMeta *v1.ObjectMeta, command, wlid string) {
	switch command {
	case REMOVE:
		removeCAMetadata(objectMeta)
	}
}

func injectPod(metadata *v1.ObjectMeta, spec *corev1.PodSpec, command, wlid string) {
	switch command {
	case UPDATE:
		injectWlid(&metadata.Annotations, wlid)
		injectTime(&metadata.Annotations)
		injectLabel(&metadata.Labels)

	case SIGN:
		updateLabel(&metadata.Labels)
		injectTime(&metadata.Annotations)
	case RESTART:
		injectTime(&metadata.Annotations)
	case REMOVE:
		restoreConatinerCommand(spec)
		removeCASpec(spec)
		removeCAMetadata(metadata)
	}
	removeIDLabels(metadata.Labels)
}

func injectNS(metadata *v1.ObjectMeta, command string) {
	switch command {
	case INJECT:
		injectTime(&metadata.Annotations)
		injectLabel(&metadata.Labels)

	case REMOVE:
		removeCAMetadata(metadata)
	}
	removeIDLabels(metadata.Labels)
}
func restoreConatinerCommand(spec *corev1.PodSpec) {
	cmdEnv := "CAA_OVERRIDDEN_CMD"
	argsEnv := "CAA_OVERRIDDEN_ARGS"
	for con := range spec.Containers {
		for env := range spec.Containers[con].Env {
			if spec.Containers[con].Env[env].Name == cmdEnv {
				cmdVal := spec.Containers[con].Env[env].Value
				if cmdVal == "nil" {
					glog.Errorf("invalid env value. conatiner: %s, env: %s=%s. current container command: %v, current container args: %v", spec.Containers[con].Name, cmdEnv, cmdVal, spec.Containers[con].Command, spec.Containers[con].Args)
					continue
				}
				newCMD := []string{}
				json.Unmarshal([]byte(cmdVal), &newCMD)
				spec.Containers[con].Command = newCMD
			}
			if spec.Containers[con].Env[env].Name == argsEnv {
				argsVal := spec.Containers[con].Env[env].Value
				if argsVal == "nil" {
					glog.Errorf("invalid env value. conatiner: %s, env: %s=%s. current container command: %v, current container args: %v", spec.Containers[con].Name, argsEnv, argsVal, spec.Containers[con].Command, spec.Containers[con].Args)
					continue
				}
				newArgs := []string{}
				json.Unmarshal([]byte(argsVal), &newArgs)
				spec.Containers[con].Args = newArgs
			}
		}
	}
}
func removeCASpec(spec *corev1.PodSpec) {
	// remove init container
	nOfContainers := len(spec.InitContainers)
	for i := 0; i < nOfContainers; i++ {
		if spec.InitContainers[i].Name == cautils.CAInitContainerName {
			if nOfContainers < 2 { //i is the only element in the slice so we need to remove this entry from the map
				spec.InitContainers = []corev1.Container{}
			} else if i == nOfContainers-1 { // i is the last element in the slice so i+1 is out of range
				spec.InitContainers = spec.InitContainers[:i]
			} else {
				spec.InitContainers = append(spec.InitContainers[:i], spec.InitContainers[i+1:]...)
			}
			nOfContainers--
			i--
		}
	}

	// remove volumes
	for injected := range cautils.InjectedVolumes {
		removeVolumes(&spec.Volumes, cautils.InjectedVolumes[injected])
	}

	// remove environment varibles
	for i := range spec.Containers {
		for injectedEnvs := range cautils.InjectedEnvironments {
			removeEnvironmentVariable(&spec.Containers[i].Env, cautils.InjectedEnvironments[injectedEnvs])
		}
	}

	// remove volumeMounts
	for i := range spec.Containers {
		for injected := range cautils.InjectedVolumeMounts {
			removeVolumeMounts(&spec.Containers[i].VolumeMounts, cautils.InjectedVolumeMounts[injected])
		}
	}
}

func removeEnvironmentVariable(envs *[]corev1.EnvVar, env string) {
	nOfEnvs := len(*envs)
	for i := 0; i < nOfEnvs; i++ {
		if (*envs)[i].Name == env {
			if nOfEnvs < 2 { //i is the only element in the slice so we need to remove this entry from the map
				*envs = []corev1.EnvVar{}
			} else if i == nOfEnvs-1 { // i is the last element in the slice so i+1 is out of range
				*envs = (*envs)[:i]
			} else {
				*envs = append((*envs)[:i], (*envs)[i+1:]...)
			}
			nOfEnvs--
			i--
		}
	}
}

func removeVolumes(volumes *[]corev1.Volume, vol string) {
	nOfvolumes := len(*volumes)
	for i := 0; i < nOfvolumes; i++ {
		if (*volumes)[i].Name == vol {
			if nOfvolumes < 2 { //i is the only element in the slice so we need to remove this entry from the map
				*volumes = []corev1.Volume{}
			} else if i == nOfvolumes-1 { // i is the last element in the slice so i+1 is out of range
				*volumes = (*volumes)[:i]
			} else {
				*volumes = append((*volumes)[:i], (*volumes)[i+1:]...)
			}
			nOfvolumes--
			i--
		}
	}
}

func removeVolumeMounts(volumes *[]corev1.VolumeMount, vol string) {
	nOfvolumes := len(*volumes)
	for i := 0; i < nOfvolumes; i++ {
		if (*volumes)[i].Name == vol {
			if nOfvolumes < 2 { //i is the only element in the slice so we need to remove this entry from the map
				*volumes = []corev1.VolumeMount{}
			} else if i == nOfvolumes-1 { // i is the last element in the slice so i+1 is out of range
				*volumes = (*volumes)[:i]
			} else {
				*volumes = append((*volumes)[:i], (*volumes)[i+1:]...)
			}
			nOfvolumes--
			i--
		}
	}
}

func injectWlid(annotations *map[string]string, wlid string) {
	if *annotations == nil {
		(*annotations) = make(map[string]string)
	}
	(*annotations)[CAWlidOld] = wlid
	(*annotations)[CAWlid] = wlid
}

func injectTime(annotations *map[string]string) {
	if *annotations == nil {
		(*annotations) = make(map[string]string)
	}
	(*annotations)[CAUpdate] = string(time.Now().UTC().Format("02-01-2006 15:04:05"))
}

func updateLabel(labels *map[string]string) {
	if *labels == nil {
		(*labels) = make(map[string]string)
	}
	(*labels)[CALabel] = "signed"
}

func injectLabel(labels *map[string]string) {
	if *labels == nil {
		(*labels) = make(map[string]string)
	}
	(*labels)[CAInject] = "add"
	(*labels)[CAInjectOld] = "add" // DEPRECATED
}

func removeCAMetadata(meatdata *v1.ObjectMeta) {
	delete(meatdata.Labels, CAInject)
	delete(meatdata.Labels, CAInjectOld) // DEPRECATED
	delete(meatdata.Labels, CALabel)
	delete(meatdata.Annotations, CAWlidOld) // DEPRECATED
	delete(meatdata.Annotations, CAStatus)
	delete(meatdata.Annotations, CASigned)
	delete(meatdata.Annotations, CAWlid)
	delete(meatdata.Annotations, CAAttached)
}

func cleanSelector(selector *v1.LabelSelector) {
	delete(selector.MatchLabels, controllerLable)
	if len(selector.MatchLabels) == 0 && len(selector.MatchLabels) == 0 {
		selector = &v1.LabelSelector{}
	}
}

func removeIDLabels(labels map[string]string) {
	delete(labels, controllerLable)
}
