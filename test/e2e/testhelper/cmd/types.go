/*
Copyright 2023 cuisongliu@qq.com.

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

package cmd

type PodStruct struct {
	Items []struct {
		ID       string `json:"id"`
		Metadata struct {
			Name      string `json:"name"`
			UID       string `json:"uid"`
			Namespace string `json:"namespace"`
			Attempt   int    `json:"attempt"`
		} `json:"metadata"`
		Labels         map[string]string `json:"labels,omitempty"`
		Annotations    map[string]string `json:"annotations,omitempty"`
		State          string            `json:"state"`
		CreatedAt      string            `json:"createdAt"`
		RuntimeHandler string            `json:"runtimeHandler"`
	} `json:"items"`
}

type ImageStruct struct {
	Images []struct {
		ID          string      `json:"id"`
		RepoTags    []string    `json:"repoTags"`
		RepoDigests []string    `json:"repoDigests"`
		Size        string      `json:"size"`
		UID         interface{} `json:"uid"`
		Username    string      `json:"username"`
		Spec        interface{} `json:"spec"`
		Pinned      bool        `json:"pinned"`
	} `json:"images"`
}

type ProcessStruct struct {
	Containers []struct {
		ID           string `json:"id"`
		PodSandboxID string `json:"podSandboxId"`
		Metadata     struct {
			Name    string `json:"name"`
			Attempt int    `json:"attempt"`
		} `json:"metadata"`
		Image struct {
			Image       string            `json:"image"`
			Annotations map[string]string `json:"annotations"`
		} `json:"image"`
		ImageRef    string            `json:"imageRef"`
		State       string            `json:"state"`
		CreatedAt   string            `json:"createdAt"`
		Labels      map[string]string `json:"labels"`
		Annotations map[string]string `json:"annotations"`
	} `json:"containers"`
}
