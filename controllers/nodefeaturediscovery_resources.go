/*

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

package controllers

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	secv1 "github.com/openshift/api/security/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/kubectl/pkg/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// assetsFromFile is a list where each item in the list contains the
// contents of a given file as a list of bytes
type assetsFromFile []byte

// Resources holds objects owned by NFD. This struct is used with the
// 'NFD' struct to assist in the process of checking if NFD's resources
// are 'Ready' or 'NotReady'.
type Resources struct {
	Namespace          corev1.Namespace
	ServiceAccount     corev1.ServiceAccount
	Role               rbacv1.Role
	RoleBinding        rbacv1.RoleBinding
	ClusterRole        rbacv1.ClusterRole
	ClusterRoleBinding rbacv1.ClusterRoleBinding
	ConfigMap          corev1.ConfigMap
	DaemonSet          appsv1.DaemonSet
	Deployment         appsv1.Deployment
	Pod                corev1.Pod
	Service            corev1.Service
}

// filePathWalkDir finds all non-directory files under the given path recursively,
// i.e. including its subdirectories
func filePathWalkDir(root string) ([]string, error) {
	var files []string

	// Walk through the files in path, and if the os.FileInfo object
	// states that the item is not a directory, append it
	// to the list of files
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

// getAssetsFrom takes a path as an input and grabs all of the
// file names in that path, then returns a list of the manifests
// it found in that path.
func getAssetsFrom(path string) []assetsFromFile {
	// All assets (manifests) as raw data
	manifests := []assetsFromFile{}
	assets := path

	// For the given path, find a list of all the files
	files, err := filePathWalkDir(assets)
	if err != nil {
		panic(err)
	}

	// For each file in the 'files' list, read the file
	// and store its contents in 'manifests'
	for _, file := range files {

		// Read the file and return its contents in
		// 'buffer'
		buffer, err := ioutil.ReadFile(file)

		// If we have an error, then something
		// unexpectedly went wrong when reading the
		// file's contents
		if err != nil {
			panic(err)
		}

		// If the reading goes smoothly, then append
		// the buffer (the file's contents) to the
		// list of manifests
		manifests = append(manifests, buffer)
	}
	return manifests
}

func addResourcesControls(path string) (Resources, controlFunc) {
	// Information about the manifest
	res := Resources{}

	// ctrl is a controlFunc object that contains a function
	// that returns information about the status of a resource
	// (i.e., Ready or NotReady)
	ctrl := controlFunc{}

	// Get the list of manifests from the given path
	manifests := getAssetsFrom(path)

	// s and reg are used later on to parse the manifest YAML
	s := json.NewYAMLSerializer(json.DefaultMetaFactory, scheme.Scheme,
		scheme.Scheme)
	reg, _ := regexp.Compile(`\b(\w*kind:\w*)\B.*\b`)

	// For each manifest, find its kind, then append the
	// appropriate function (e.g., 'Namespace' or 'Role') to
	// ctrl so that the Namespace, Role, etc. can be parsed
	for _, m := range manifests {
		kind := reg.FindString(string(m))
		slce := strings.Split(kind, ":")
		kind = strings.TrimSpace(slce[1])

		switch kind {
		case "Namespace":
			_, _, err := s.Decode(m, nil, &res.Namespace)
			panicIfError(err)
			ctrl = append(ctrl, Namespace)
		case "ServiceAccount":
			_, _, err := s.Decode(m, nil, &res.ServiceAccount)
			panicIfError(err)
			ctrl = append(ctrl, ServiceAccount)
		case "ClusterRole":
			_, _, err := s.Decode(m, nil, &res.ClusterRole)
			panicIfError(err)
			ctrl = append(ctrl, ClusterRole)
		case "ClusterRoleBinding":
			_, _, err := s.Decode(m, nil, &res.ClusterRoleBinding)
			panicIfError(err)
			ctrl = append(ctrl, ClusterRoleBinding)
		case "Role":
			_, _, err := s.Decode(m, nil, &res.Role)
			panicIfError(err)
			ctrl = append(ctrl, Role)
		case "RoleBinding":
			_, _, err := s.Decode(m, nil, &res.RoleBinding)
			panicIfError(err)
			ctrl = append(ctrl, RoleBinding)
		case "ConfigMap":
			_, _, err := s.Decode(m, nil, &res.ConfigMap)
			panicIfError(err)
			ctrl = append(ctrl, ConfigMap)
		case "DaemonSet":
			_, _, err := s.Decode(m, nil, &res.DaemonSet)
			panicIfError(err)
			ctrl = append(ctrl, DaemonSet)
		case "Deployment":
			_, _, err := s.Decode(m, nil, &res.Deployment)
			panicIfError(err)
			ctrl = append(ctrl, Deployment)
		case "Service":
			_, _, err := s.Decode(m, nil, &res.Service)
			panicIfError(err)
			ctrl = append(ctrl, Service)
		case "SecurityContextConstraints":
			_, _, err := s.Decode(m, nil, &res.SecurityContextConstraints)
			panicIfError(err)
			ctrl = append(ctrl, SecurityContextConstraints)

		default:
			r.Log.Infof("Unknown Resource: ", "Kind", kind)
		}

	}

	return res, ctrl
}

// Trigger a panic if an error occurs
func panicIfError(err error) {
	if err != nil {
		panic(err)
	}
}

// getServiceAccount gets one of the NFD Operand's ServiceAccounts
func (r *NodeFeatureDiscoveryReconciler) getServiceAccount(ctx context.Context, namespace string, name string) (*corev1.ServiceAccount, error) {
	sa := &corev1.ServiceAccount{}
	err := r.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, sa)
	return sa, err
}

// getDaemonSet gets one of the NFD Operand's DaemonSets
func (r *NodeFeatureDiscoveryReconciler) getDaemonSet(ctx context.Context, namespace string, name string) (*appsv1.DaemonSet, error) {
	ds := &appsv1.DaemonSet{}
	err := r.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, ds)
	return ds, err
}

// getConfigMap gets one of the NFD Operand's ConfigMap
func (r *NodeFeatureDiscoveryReconciler) getConfigMap(ctx context.Context, namespace string, name string) (*corev1.ConfigMap, error) {
	cm := &corev1.ConfigMap{}
	err := r.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, cm)
	return cm, err
}

// getService gets one of the NFD Operand's Services
func (r *NodeFeatureDiscoveryReconciler) getService(ctx context.Context, namespace string, name string) (*corev1.Service, error) {
	svc := &corev1.Service{}
	err := r.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, svc)
	return svc, err
}

// getRole gets one of the NFD Operand's Roles
func (r *NodeFeatureDiscoveryReconciler) getRole(ctx context.Context, namespace string, name string) (*rbacv1.Role, error) {
	role := &rbacv1.Role{}
	err := r.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, role)
	return role, err
}

// getRoleBinding gets one of the NFD Operand's RoleBindings
func (r *NodeFeatureDiscoveryReconciler) getRoleBinding(ctx context.Context, namespace string, name string) (*rbacv1.RoleBinding, error) {
	rb := &rbacv1.RoleBinding{}
	err := r.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, rb)
	return rb, err
}

// getClusterRole gets one of the NFD Operand's ClusterRoles
func (r *NodeFeatureDiscoveryReconciler) getClusterRole(ctx context.Context, namespace string, name string) (*rbacv1.ClusterRole, error) {
	cr := &rbacv1.ClusterRole{}
	err := r.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, cr)
	return cr, err
}

// getClusterRoleBinding gets one of the NFD Operand's ClusterRoleBindings
func (r *NodeFeatureDiscoveryReconciler) getClusterRoleBinding(ctx context.Context, namespace string, name string) (*rbacv1.ClusterRoleBinding, error) {
	crb := &rbacv1.ClusterRoleBinding{}
	err := r.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, crb)
	return crb, err
}

// deleteServiceAccount deletes one of the NFD Operand's ServiceAccounts
func (r *NodeFeatureDiscoveryReconciler) deleteServiceAccount(ctx context.Context, namespace string, name string) error {
	sa, err := r.getServiceAccount(ctx, namespace, name)

	// Do not return an error if the object has already been deleted
	if k8serrors.IsNotFound(err) {
		return nil
	}

	if err != nil {
		return err
	}

	return r.Delete(context.TODO(), sa)
}

// deleteConfigMap deletes the NFD Operand ConfigMap
func (r *NodeFeatureDiscoveryReconciler) deleteConfigMap(ctx context.Context, namespace string, name string) error {
	cm, err := r.getConfigMap(ctx, namespace, name)

	// Do not return an error if the object has already been deleted
	if k8serrors.IsNotFound(err) {
		return nil
	}

	if err != nil {
		return err
	}

	return r.Delete(context.TODO(), cm)
}

// deleteDaemonSet deletes Operand DaemonSet
func (r *NodeFeatureDiscoveryReconciler) deleteDaemonSet(ctx context.Context, namespace string, name string) error {
	ds, err := r.getDaemonSet(ctx, namespace, name)

	// Do not return an error if the object has already been deleted
	if k8serrors.IsNotFound(err) {
		return nil
	}

	if err != nil {
		return err
	}

	return r.Delete(context.TODO(), ds)
}

// deleteService deletes the NFD Operand's Service
func (r *NodeFeatureDiscoveryReconciler) deleteService(ctx context.Context, namespace string, name string) error {
	svc, err := r.getService(ctx, namespace, name)

	// Do not return an error if the object has already been deleted
	if k8serrors.IsNotFound(err) {
		return nil
	}

	if err != nil {
		return err
	}

	return r.Delete(context.TODO(), svc)
}

// deleteRole deletes one of the NFD Operand's Roles
func (r *NodeFeatureDiscoveryReconciler) deleteRole(ctx context.Context, namespace string, name string) error {
	role, err := r.getRole(ctx, namespace, name)

	// Do not return an error if the object has already been deleted
	if k8serrors.IsNotFound(err) {
		return nil
	}

	if err != nil {
		return err
	}

	return r.Delete(context.TODO(), role)
}

// deleteRoleBinding deletes one of the NFD Operand's RoleBindings
func (r *NodeFeatureDiscoveryReconciler) deleteRoleBinding(ctx context.Context, namespace string, name string) error {
	rb, err := r.getRoleBinding(ctx, namespace, name)

	// Do not return an error if the object has already been deleted
	if k8serrors.IsNotFound(err) {
		return nil
	}

	if err != nil {
		return err
	}

	return r.Delete(context.TODO(), rb)
}

// deleteClusterRole deletes one of the NFD Operand's ClusterRoles
func (r *NodeFeatureDiscoveryReconciler) deleteClusterRole(ctx context.Context, namespace string, name string) error {
	cr, err := r.getClusterRole(ctx, namespace, name)

	// Do not return an error if the object has already been deleted
	if k8serrors.IsNotFound(err) {
		return nil
	}

	if err != nil {
		return err
	}

	return r.Delete(context.TODO(), cr)
}

// deleteClusterRoleBinding deletes one of the NFD Operand's ClusterRoleBindings
func (r *NodeFeatureDiscoveryReconciler) deleteClusterRoleBinding(ctx context.Context, namespace string, name string) error {
	crb, err := r.getClusterRoleBinding(ctx, namespace, name)

	// Do not return an error if the object has already been deleted
	if k8serrors.IsNotFound(err) {
		return nil
	}

	if err != nil {
		return err
	}

	return r.Delete(context.TODO(), crb)
}
