package notificationhandler

import (
	"encoding/json"
	"fmt"
	"k8s-ca-websocket/cautils"
	"net/url"
	"strings"
	"time"

	"github.com/armosec/armoapi-go/apis"
	"github.com/armosec/armoapi-go/armotypes"
	"github.com/armosec/cluster-notifier-api-go/notificationserver"
	opapolicy "github.com/armosec/opa-utils/reporthandling"
	"gopkg.in/mgo.v2/bson"

	"github.com/golang/glog"
)

func (notification *NotificationHandler) websocketPingMessage() error {
	for {
		time.Sleep(30 * time.Second)
		if err := notification.connector.WritePingMessage(); err != nil {
			glog.Errorf("PING, %s", err.Error())
			return fmt.Errorf("PING, %s", err.Error())
		}
	}
}

func (notification *NotificationHandler) handleJsonNotification(bytesNotification []byte) error {
	notif := &notificationserver.Notification{}
	if err := json.Unmarshal(bytesNotification, notif); err != nil {
		glog.Error(err)
		return err
	}

	dst := notif.Target["dest"]
	switch dst {
	case "kubescape":
		// sent by this function in dash BE: KubescapeInClusterHandler
		// in file: httphandlerv2/posturehandler.go
		kubescapeNotification, err := convertJsonNotification(bytesNotification)
		if err != nil {
			return fmt.Errorf("handleJsonNotification")
		}

		sessionOnj := cautils.NewSessionObj(&apis.Command{
			CommandName: string(kubescapeNotification.NotificationType),
			Designators: []armotypes.PortalDesignator{kubescapeNotification.Designators},
			JobTracking: apis.JobTracking{JobID: kubescapeNotification.JobID},
			Args:        map[string]interface{}{"rules": kubescapeNotification.Rules},
		}, "WebSocket", "", kubescapeNotification.JobID, 1)
		*notification.sessionObj <- *sessionOnj
	case "", "safeMode":
		safeMode, e := parseSafeModeNotification(notif.Notification)
		if e != nil {
			return e
		}

		// send to pipe
		*notification.safeModeObj <- *safeMode
	}

	return nil

}
func convertBsonNotification(bytesNotification []byte) (*apis.SafeMode, error) {
	notification := &notificationserver.Notification{}
	if err := bson.Unmarshal(bytesNotification, notification); err != nil {
		if err := json.Unmarshal(bytesNotification, notification); err != nil {
			glog.Error(err)
			return nil, err
		}
	}

	safeMode := apis.SafeMode{}
	notificationBytes, ok := notification.Notification.([]byte)
	if !ok {
		var err error
		notificationBytes, err = json.Marshal(notification.Notification)
		if err != nil {
			return &safeMode, err
		}
	}

	glog.Infof("Notification: %s\n", string(notificationBytes))
	if err := json.Unmarshal(notificationBytes, &safeMode); err != nil {
		glog.Error(err)
		return nil, err
	}
	return &safeMode, nil

}

func initARMOHelmNotificationServiceURL() string {
	urlObj := url.URL{}
	host := cautils.NotificationServerWSURL
	if host == "" {
		return ""
	}

	scheme := "ws"
	if strings.HasPrefix(host, "ws://") {
		host = strings.TrimPrefix(host, "ws://")
		scheme = "ws"
	} else if strings.HasPrefix(host, "wss://") {
		host = strings.TrimPrefix(host, "wss://")
		scheme = "wss"
	}

	urlObj.Scheme = scheme
	urlObj.Host = host
	urlObj.Path = notificationserver.PathWebsocketV1

	q := urlObj.Query()
	q.Add(notificationserver.TargetCustomer, cautils.ClusterConfig.CustomerGUID)
	q.Add(notificationserver.TargetCluster, cautils.ClusterConfig.ClusterName)
	q.Add(notificationserver.TargetComponent, notificationserver.TargetComponentTriggerHandler)
	urlObj.RawQuery = q.Encode()

	return urlObj.String()
}

func initNotificationServerURL() string {
	urlObj := url.URL{}
	host := cautils.NotificationServerWSURL
	if host == "" {
		return ""
	}

	scheme := "ws"
	if strings.HasPrefix(host, "ws://") {
		host = strings.TrimPrefix(host, "ws://")
		scheme = "ws"
	} else if strings.HasPrefix(host, "wss://") {
		host = strings.TrimPrefix(host, "wss://")
		scheme = "wss"
	}

	urlObj.Scheme = scheme
	urlObj.Host = host
	urlObj.Path = notificationserver.PathWebsocketV1

	q := urlObj.Query()
	// customerGUID := strings.ToUpper(cautils.CustomerGUID)
	// customerGUID = strings.Replace(customerGUID, "-", "", -1)
	// q.Add(notificationserver.TargetCustomer, customerGUID)
	// q.Add(notificationserver.TargetCluster, cautils.ClusterName)
	q.Add(notificationserver.TargetComponent, notificationserver.TargetComponentLoggerValue)
	q.Add(notificationserver.TargetComponent, notificationserver.TargetComponentTriggerHandler)
	urlObj.RawQuery = q.Encode()

	return urlObj.String()
}

func parseSafeModeNotification(notification interface{}) (*apis.SafeMode, error) {
	safeMode := &apis.SafeMode{}
	notificationBytes, err := json.Marshal(notification)
	if err != nil {
		return safeMode, err
	}

	glog.Infof("Notification: %s", string(notificationBytes))
	if err := json.Unmarshal(notificationBytes, safeMode); err != nil {
		glog.Error(err)
		return safeMode, err
	}
	if safeMode.InstanceID == "" {
		safeMode.InstanceID = safeMode.PodName
	}

	return safeMode, nil
}

func convertJsonNotification(bytesNotification []byte) (*opapolicy.PolicyNotification, error) {
	notification := &notificationserver.Notification{}
	if err := json.Unmarshal(bytesNotification, notification); err != nil {
		glog.Error(err)
		return nil, err
	}
	policyNotificationBytes, ok := notification.Notification.([]byte)
	if !ok {
		err := fmt.Errorf("Failed converting notification to []byte")
		glog.Error(err)
		return nil, err
	}
	glog.Infof("Notification: %s", string(policyNotificationBytes))
	policyNotification := &opapolicy.PolicyNotification{}
	if err := json.Unmarshal(policyNotificationBytes, policyNotification); err != nil {
		glog.Error(err)
		return nil, err
	}
	return policyNotification, nil

}
