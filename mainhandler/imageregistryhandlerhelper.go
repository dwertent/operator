package mainhandler

import (
	"context"
	"fmt"
	"k8s-ca-websocket/cautils"
	"math/rand"
	"time"

	"github.com/golang/glog"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (actionHandler *ActionHandler) updateRegistryScanCronJob() error {
	jobParams := actionHandler.command.GetCronJobParams()
	if jobParams == nil {
		glog.Infof("updateRegistryScanCronJob: failed to get jobParams")
		return fmt.Errorf("failed to get failed to get jobParams")
	}

	jobTemplateObj, err := actionHandler.k8sAPI.KubernetesClient.BatchV1().CronJobs(cautils.CA_NAMESPACE).Get(context.Background(), jobParams.JobName, metav1.GetOptions{})
	if err != nil {
		glog.Infof("updateRegistryScanCronJob: failed to get cronjob: %s", jobParams.JobName)
		return err
	}

	jobTemplateObj.Spec.Schedule = getCronTabSchedule(actionHandler.command)
	if jobTemplateObj.Spec.JobTemplate.Spec.Template.Annotations == nil {
		jobTemplateObj.Spec.JobTemplate.Spec.Template.Annotations = make(map[string]string)
	}

	jobTemplateObj.Spec.JobTemplate.Spec.Template.Annotations[armoJobIDAnnotation] = actionHandler.command.JobTracking.JobID

	_, err = actionHandler.k8sAPI.KubernetesClient.BatchV1().CronJobs(cautils.CA_NAMESPACE).Update(context.Background(), jobTemplateObj, metav1.UpdateOptions{})
	if err != nil {
		return err
	}
	glog.Infof("updateRegistryScanCronJob: cronjob: %v updated successfully", jobParams.JobName)
	return nil

}

func (actionHandler *ActionHandler) setRegistryScanCronJob(sessionObj *cautils.SessionObj) error {
	registryScanHandler := NewRegistryScanHandler()

	// parse registry name from command
	registryName, err := actionHandler.parseRegistryNameArg(sessionObj)
	if err != nil {
		glog.Infof("setRegistryScanCronJob: error parsing registry name from command: %s", err.Error())
		return err
	}

	// name is registryScanConfigmap name + random string - configmap and cronjob
	name := fixK8sCronJobNameLimit(fmt.Sprintf("%d-%s", rand.NewSource(time.Now().UnixNano()).Int63(), registryScanConfigmap))

	// create configmap with POST data to trigger websocket
	err = registryScanHandler.createTriggerRequestConfigMap(actionHandler.k8sAPI, name, registryName, sessionObj.Command)
	if err != nil {
		glog.Infof("setRegistryScanCronJob: error creating configmap : %s", err.Error())
		return err
	}

	err = registryScanHandler.createTriggerRequestCronJob(actionHandler.k8sAPI, name, registryName, sessionObj.Command)
	if err != nil {
		glog.Infof("setRegistryScanCronJob: error creating conjob : %s", err.Error())
		return err
	}

	glog.Infof("setRegistryScanCronJob: cronjob: %s created successfully", name)
	return err
}

func (actionHandler *ActionHandler) deleteRegistryScanCronJob() error {
	jobParams := actionHandler.command.GetCronJobParams()
	if jobParams == nil {
		glog.Infof("updateRegistryScanCronJob: failed to get jobParams")
		return fmt.Errorf("failed to get jobParams")
	}

	return actionHandler.deleteCronjob(jobParams.JobName)
}
