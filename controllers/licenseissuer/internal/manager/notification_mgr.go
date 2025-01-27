/*
Copyright 2023.

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

package manager

import (
	"context"
	"crypto/rand"
	"io"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-logr/logr"
	ntf "github.com/labring/sealos/controllers/common/notification/api/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	cl "sigs.k8s.io/controller-runtime/pkg/client"
)

type Target string

const (
	UserPrefix string = "ns-"
	AdmPrefix  string = "adm-"
)

type NotificationManager struct {
	TimeLastPull int64
}

func (nm *NotificationManager) InitTime() {
	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	nm.TimeLastPull = startOfDay.Unix()
}

type NotificationPackage struct {
	Name    string
	Title   Title
	From    Source
	Message Message
	Level   ntf.Type
}

const (
	idLength    = 12
	letterBytes = "abcdefghijklmnopqrstuvwxyz0123456789"
)

func randStringBytes(n int) (string, error) {
	bytes := make([]byte, n)
	if _, err := io.ReadFull(rand.Reader, bytes); err != nil {
		return "", err
	}
	for i, b := range bytes {
		bytes[i] = letterBytes[b%byte(len(letterBytes))]
	}
	return string(bytes), nil
}

func NewNotificationPackage(title Title, from Source, message Message) NotificationPackage {
	id, err := randStringBytes(idLength)
	if err != nil || id == "" {
		id = strings.ToLower(strconv.Itoa(int(time.Now().Unix())))
	}
	return NotificationPackage{
		Name:    id,
		Title:   title,
		From:    from,
		Message: message,
	}
}

func NewNotificationPackageWithLevel(title Title, from Source, message Message, level ntf.Type) NotificationPackage {
	id, err := randStringBytes(idLength)
	if err != nil || id == "" {
		id = strings.ToLower(strconv.Itoa(int(time.Now().Unix())))
	}
	return NotificationPackage{
		Name:    id,
		Title:   title,
		From:    from,
		Message: message,
		Level:   level,
	}
}

type NotificationTask struct {
	ctx               context.Context
	client            cl.Client
	Target            string
	NotificationCache []ntf.Notification
}

func NewNotificationTask(ctx context.Context, client cl.Client, target string, cache []ntf.Notification) NotificationTask {
	return NotificationTask{
		ctx:               ctx,
		client:            client,
		Target:            target,
		NotificationCache: cache,
	}
}

func NotificationPackageToNotification(pack NotificationPackage) ntf.Notification {
	var notification ntf.Notification
	notification.Name = pack.Name
	notification.Spec.Timestamp = time.Now().Unix()
	notification.Spec.From = string(pack.From)
	notification.Spec.Message = string(pack.Message)
	notification.Spec.Title = string(pack.Title)
	notification.Spec.Importance = pack.Level
	return notification
}

func (nt *NotificationTask) Run() error {
	for _, data := range nt.NotificationCache {
		data.Namespace = nt.Target
		var tmp ntf.Notification
		err := nt.client.Get(nt.ctx, types.NamespacedName{Namespace: data.Namespace, Name: data.Name}, &tmp)
		if err != nil && apierrors.IsNotFound(err) {
			if err = nt.client.Create(nt.ctx, &data); err != nil {
				return err
			}
			continue
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func NewNotificationManager() *NotificationManager {
	return &NotificationManager{
		TimeLastPull: time.Now().Add(-14 * 24 * time.Hour).Unix(),
	}
}

func NotificationResponseToNotification(resp NotificationResponse) ntf.Notification {
	var res ntf.Notification
	res.Name = resp.ID
	res.Spec.Title = resp.Title
	res.Spec.Message = resp.Message
	res.Spec.Timestamp = time.Now().Unix()
	res.Spec.From = "Sealos Cloud"
	res.SetGroupVersionKind(schema.GroupVersionKind{Group: "notification.sealos.io", Version: "v1", Kind: "Notification"})
	return res
}

type Set interface {
	Add(item string)
	Remove(item string)
	Contains(item string) bool
	Iter() <-chan string
}

type StringSet struct {
	set map[string]struct{}
}

func NewStringSet() *StringSet {
	return &StringSet{set: make(map[string]struct{})}
}

func (s *StringSet) Add(item string) {
	s.set[item] = struct{}{}
}

func (s *StringSet) Remove(item string) {
	delete(s.set, item)
}

func (s *StringSet) Contains(item string) bool {
	_, exists := s.set[item]
	return exists
}

func (s *StringSet) Iter() <-chan string {
	ch := make(chan string)
	go func() {
		for item := range s.set {
			ch <- item
		}
		close(ch)
	}()
	return ch
}

type UserCategory map[string]Set

type Users interface {
	Add(prefix string, nsName string)
}

func (uc UserCategory) Add(prefix string, nsName string) {
	if strings.HasPrefix(nsName, prefix) {
		if _, exists := uc[prefix]; !exists {
			uc[prefix] = NewStringSet()
		}
		uc[prefix].Add(nsName)
	}
}

func (uc *UserCategory) GetNameSpace(ctx context.Context, client cl.Client) error {
	nsList := &corev1.NamespaceList{}
	if err := client.List(ctx, nsList); err != nil {
		return err
	}
	for _, ns := range nsList.Items {
		uc.Add(UserPrefix, ns.Name)
		uc.Add(AdmPrefix, ns.Name)
	}
	return nil
}

var noticeLogger logr.Logger

func init() {
	noticeLogger = ctrl.Log.WithName("SubNotification")
}

// SubmitNotificationWithUserCategory is a function that submits a notification to a group of users specified by a category.
// It creates a new goroutine for each user in the category to concurrently send notifications.
// It will log any errors occurred during the process.
func SubmitNotificationWithUserCategory(ctx context.Context, client cl.Client, users UserCategory, prefix string, pack NotificationPackage) {
	notification := NotificationPackageToNotification(pack)
	var wg sync.WaitGroup
	errchan := make(chan error)
	for ns := range users[prefix].Iter() {
		wg.Add(1)
		notificationTask := NewNotificationTask(ctx, client, ns, []ntf.Notification{notification})
		go AsyncCloudTask(&wg, errchan, &notificationTask)
	}
	go func() {
		wg.Wait()
		close(errchan)
	}()
	for err := range errchan {
		if err != nil {
			noticeLogger.Error(err, "Failed to deliver notification success.")
		}
	}
}

// SubmitNotificationWithUser is a function that submits a notification to a single user.
// It creates a new goroutine to send the notification and logs any errors occurred during the process.
func SubmitNotificationWithUser(ctx context.Context, client cl.Client, target string, pack NotificationPackage) {
	notification := NotificationPackageToNotification(pack)
	notificationTask := NewNotificationTask(ctx, client, target, []ntf.Notification{notification})
	var wg sync.WaitGroup
	errchan := make(chan error)
	wg.Add(1)
	go AsyncCloudTask(&wg, errchan, &notificationTask)
	go func() {
		wg.Wait()
		close(errchan)
	}()
	for err := range errchan {
		if err != nil {
			noticeLogger.Error(err, "Failed to deliver notification success")
		}
	}
}
