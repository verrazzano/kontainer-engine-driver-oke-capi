// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package provisioning

import (
	"context"
	"fmt"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes"
	"strings"
	"time"
)

type (
	Logger struct {
		ctx         context.Context
		ki          kubernetes.Interface
		clusterName string
	}

	// Levels are convention from provisioning-log
	level string
)

const (
	INFO  level = "[INFO]"
	ERROR level = "[ERROR]"

	// These names conventions are used by kontainer-engine for provisioning-log
	configMapName = "provisioning-log"
	logField      = "log"
	lastLogField  = "last"
	maxLength     = 10000

	clusterStatusMissingField = "loading cluster conditions"
	statusFalse               = "False"
)

var severities = []string{
	"Info",
	"Warning",
	"Error",
}

func NewLogger(ctx context.Context, ki kubernetes.Interface, cluster string) *Logger {
	return &Logger{
		ctx:         ctx,
		ki:          ki,
		clusterName: cluster,
	}
}

// ClusterStatus logs a message based on the cluster's conditions
func (l *Logger) ClusterStatus(cl *unstructured.Unstructured) error {
	conditions, ok, err := unstructured.NestedSlice(cl.Object, "status", "conditions")
	if !ok || err != nil {
		return l.Infof("initializing cluster infrastructure")
	}

	messages := map[string][]string{}
	for _, condition := range conditions {
		// condition should always be a map
		c, ok := condition.(map[string]interface{})
		if !ok {
			return l.Errorf("unexpected cluster condition: %v", c)
		}
		severity, severityOk, err := unstructured.NestedString(c, "severity")
		if err != nil {
			return l.Infof(clusterStatusMissingField)
		}
		status, _, err := unstructured.NestedString(c, "status")
		if err != nil {
			return l.Infof(clusterStatusMissingField)
		}
		// not a notable message if no severity or ready status
		if !severityOk || status != statusFalse {
			continue
		}
		message, messageOk, err := unstructured.NestedString(c, "message")
		if err != nil {
			return l.Infof(clusterStatusMissingField)
		}
		if !messageOk {
			message, messageOk, err = unstructured.NestedString(c, "reason")
			if err != nil || !messageOk {
				return l.Infof(clusterStatusMissingField)
			}
		}

		if !contains(messages[severity], message) {
			messages[severity] = append(messages[severity], message)
		}
	}

	return l.Infof(buildClusterStatusMessage(messages))
}

func contains(s []string, e string) bool {
	for _, i := range s {
		if i == e {
			return true
		}
	}
	return false
}

func buildClusterStatusMessage(messages map[string][]string) string {
	const sevDelimiter = ": "
	sb := strings.Builder{}
	for _, s := range severities {
		sm := messages[s]
		for i, m := range sm {
			sb.WriteString(m)
			if i < len(sm)-1 {
				sb.WriteString(", ")
			}
		}
		sb.WriteString(sevDelimiter)
	}
	return strings.Trim(sb.String(), sevDelimiter)
}

func (l *Logger) Infof(format string, args ...any) error {
	return l.write(INFO, fmt.Sprintf(format, args...))
}

func (l *Logger) Errorf(format string, args ...any) error {
	return l.write(ERROR, fmt.Sprintf(format, args...))
}

func (l *Logger) write(logLevel level, msg string) error {
	msg = fmt.Sprintf("%s %s %s\n", time.Now().Format(time.RFC3339), logLevel, msg)
	cm, err := l.ki.CoreV1().ConfigMaps(l.clusterName).Get(l.ctx, configMapName, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return l.createLog(msg)
	}
	if err != nil {
		return err
	}
	return l.appendLog(cm, msg)
}

func (l *Logger) createLog(msg string) error {
	_, err := l.ki.CoreV1().ConfigMaps(l.clusterName).Create(l.ctx, &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      configMapName,
			Namespace: l.clusterName,
		},
		Data: map[string]string{
			logField:     msg,
			lastLogField: msg,
		},
	}, metav1.CreateOptions{})
	return err
}

func (l *Logger) appendLog(cm *corev1.ConfigMap, msg string) error {
	if len(cm.Data) < 1 {
		cm.Data = map[string]string{}
	}
	log := cm.Data[logField]
	last := cm.Data[lastLogField]
	// Trim to last message if over length
	if len(log) >= maxLength {
		log = log[maxLength:]
		_ = strings.TrimRightFunc(log, func(r rune) bool {
			return r != '\n'
		})
	}

	if len(last) > 1 && areEquivalentMessages(msg, last) {
		return nil
	}
	cm.Data[logField] = log + msg
	cm.Data[lastLogField] = msg
	_, err := l.ki.CoreV1().ConfigMaps(l.clusterName).Update(l.ctx, cm, metav1.UpdateOptions{})
	return err
}

func areEquivalentMessages(m1, m2 string) bool {
	// Strip message timestamp before comparison
	return stripFirstWordInString(m1) == stripFirstWordInString(m2)
}

func stripFirstWordInString(s string) string {
	for i := 0; i < len(s); i++ {
		if s[i] == ' ' {
			return s[i:]
		}
	}
	return s
}
