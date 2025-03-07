/*
Copyright 2022 The KubeVela Authors.

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

package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/emicklei/go-restful/v3"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/oam-dev/kubevela/apis/types"
	"github.com/oam-dev/kubevela/pkg/auth"
	"github.com/oam-dev/kubevela/pkg/utils"

	"github.com/kubevela/velaux/pkg/server/domain/model"
	"github.com/kubevela/velaux/pkg/server/domain/repository"
	"github.com/kubevela/velaux/pkg/server/infrastructure/datastore"
	assembler "github.com/kubevela/velaux/pkg/server/interfaces/api/assembler/v1"
	apisv1 "github.com/kubevela/velaux/pkg/server/interfaces/api/dto/v1"
	apiserverutils "github.com/kubevela/velaux/pkg/server/utils"
	"github.com/kubevela/velaux/pkg/server/utils/bcode"
)

// resourceActions all register resources and actions
var resourceActions map[string][]string
var lock sync.Mutex
var reg = regexp.MustCompile(`(?U)\{.*\}`)

var defaultProjectPermissionTemplate = []*model.PermissionTemplate{
	{
		Name:  "project-view",
		Alias: "Project View",
		Resources: []string{
			"project:{projectName}",
			"project:{projectName}/config:*",
			"project:{projectName}/provider:*",
			"project:{projectName}/role:*",
			"project:{projectName}/projectUser:*",
			"project:{projectName}/permission:*",
			"project:{projectName}/environment:*",
			"project:{projectName}/application:*/*",
			"project:{projectName}/pipeline:*/*",
		},
		Actions: []string{"detail", "list"},
		Effect:  "Allow",
		Scope:   "project",
	},
	{
		Name:      "app-management",
		Alias:     "App Management",
		Resources: []string{"project:{projectName}/application:*/*"},
		Actions:   []string{"*"},
		Effect:    "Allow",
		Scope:     "project",
	},
	{
		Name:      "env-management",
		Alias:     "Environment Management",
		Resources: []string{"project:{projectName}/environment:*"},
		Actions:   []string{"*"},
		Effect:    "Allow",
		Scope:     "project",
	},
	{
		Name:      "role-management",
		Alias:     "Role Management",
		Resources: []string{"project:{projectName}/role:*", "project:{projectName}/projectUser:*", "project:{projectName}/permission:*"},
		Actions:   []string{"*"},
		Effect:    "Allow",
		Scope:     "project",
	},
	{
		Name:      "config-management",
		Alias:     "Config Management",
		Resources: []string{"project:{projectName}/config:*", "project:{projectName}/provider:*"},
		Actions:   []string{"*"},
		Effect:    "Allow",
		Scope:     "project",
	},
	{
		Name:  "pipeline-management",
		Alias: "Pipeline Management",
		Resources: []string{
			"project:{projectName}/pipeline:*/*",
		},
		Actions: []string{"*"},
		Effect:  "Allow",
		Scope:   "project",
	},
}

var defaultPlatformPermission = []*model.PermissionTemplate{
	{
		Name:      "disable-cloudshell",
		Alias:     "Disable CloudShell",
		Resources: []string{"cloudshell"},
		Actions:   []string{"*"},
		Effect:    "Deny",
		Scope:     "platform",
	},
	{
		Name:      "cluster-management",
		Alias:     "Cluster Management",
		Resources: []string{"cluster:*/*"},
		Actions:   []string{"*"},
		Effect:    "Allow",
		Scope:     "platform",
	},
	{
		Name:      "project-management",
		Alias:     "Project Management",
		Resources: []string{"project:*"},
		Actions:   []string{"*"},
		Effect:    "Allow",
		Scope:     "platform",
	},
	{
		Name:      "project-list",
		Alias:     "Project List",
		Resources: []string{"project:*"},
		Actions:   []string{"list"},
		Effect:    "Allow",
		Scope:     "platform",
	},
	{
		Name:      "addon-management",
		Alias:     "Addon Management",
		Resources: []string{"addon:*", "addonRegistry:*"},
		Actions:   []string{"*"},
		Effect:    "Allow",
		Scope:     "platform",
	},
	{
		Name:      "target-management",
		Alias:     "Target Management",
		Resources: []string{"target:*", "cluster:*/namespace:*"},
		Actions:   []string{"*"},
		Effect:    "Allow",
		Scope:     "platform",
	},
	{
		Name:      "user-management",
		Alias:     "User Management",
		Resources: []string{"user:*"},
		Actions:   []string{"*"},
		Effect:    "Allow",
		Scope:     "platform",
	},
	{
		Name:      "role-management",
		Alias:     "Platform Role Management",
		Resources: []string{"role:*", "permission:*"},
		Actions:   []string{"*"},
		Effect:    "Allow",
		Scope:     "platform",
	},
	{
		Name:      "config-management",
		Alias:     "Config Management",
		Resources: []string{"config:*/*"},
		Actions:   []string{"*"},
		Effect:    "Allow",
		Scope:     "platform",
	},
	{
		Name:      "admin",
		Alias:     "Admin",
		Resources: []string{"*"},
		Actions:   []string{"*"},
		Effect:    "Allow",
		Scope:     "platform",
	},
}

// ResourceMaps all resources definition for RBAC
var ResourceMaps = map[string]resourceMetadata{
	"project": {
		subResources: map[string]resourceMetadata{
			"application": {
				pathName: "appName",
				subResources: map[string]resourceMetadata{
					"component": {
						subResources: map[string]resourceMetadata{
							"trait": {
								pathName: "traitType",
							},
						},
						pathName: "compName",
					},
					"workflow": {
						subResources: map[string]resourceMetadata{
							"record": {
								pathName: "record",
							},
						},
						pathName: "workflowName",
					},
					"policy": {
						pathName: "policyName",
					},
					"revision": {
						pathName: "revision",
					},
					"envBinding": {
						pathName: "envName",
					},
					"trigger": {},
				},
			},
			"environment": {
				pathName: "envName",
			},
			"workflow": {
				pathName: "workflowName",
			},
			"role": {
				pathName: "roleName",
			},
			"permission": {},
			"projectUser": {
				pathName: "userName",
			},
			"applicationTemplate": {},
			"config": {
				pathName: "configName",
			},
			"provider": {},
			"pipeline": {
				pathName: "pipelineName",
				subResources: map[string]resourceMetadata{
					"context": {
						pathName: "contextName",
					},
					"pipelineRun": {
						pathName: "pipelineRunName",
					},
				},
			},
		},
		pathName: "projectName",
	},
	"cluster": {
		pathName: "clusterName",
		subResources: map[string]resourceMetadata{
			"namespace": {},
		},
	},
	"addon": {
		pathName: "addonName",
	},
	"addonRegistry": {
		pathName: "addonRegName",
	},
	"target": {
		pathName: "targetName",
	},
	"user": {
		pathName: "userName",
	},
	"role": {},
	"permission": {
		pathName: "permissionName",
	},
	"systemSetting": {},
	"definition": {
		pathName: "definitionName",
	},
	"configType": {
		pathName: "configType",
		subResources: map[string]resourceMetadata{
			"config": {
				pathName: "name",
			},
		},
	},
	"cloudshell":     {},
	"config":         {},
	"configTemplate": {},
}

var existResourcePaths = convertSources(ResourceMaps)

type resourceMetadata struct {
	subResources map[string]resourceMetadata
	pathName     string
}

func checkResourcePath(resource string) (string, error) {
	if sub, exist := ResourceMaps[resource]; exist {
		if sub.pathName != "" {
			return fmt.Sprintf("%s:{%s}", resource, sub.pathName), nil
		}
		return fmt.Sprintf("%s:*", resource), nil
	}
	path := ""
	exist := 0
	lastResourceName := resource[strings.LastIndex(resource, "/")+1:]
	for key, erp := range existResourcePaths {
		allMatchIndex := strings.Index(key, fmt.Sprintf("/%s/", resource))
		index := strings.Index(erp, fmt.Sprintf("/%s:", lastResourceName))
		if index > -1 && allMatchIndex > -1 {
			pre := erp[:index+len(lastResourceName)+2]
			next := strings.Replace(erp, pre, "", 1)
			nameIndex := strings.Index(next, "/")
			if nameIndex > -1 {
				pre += next[:nameIndex]
			}
			if pre != path {
				exist++
			}
			path = pre
		}
	}
	path = strings.TrimSuffix(path, "/")
	path = strings.TrimPrefix(path, "/")
	if exist == 1 {
		return path, nil
	}
	if exist > 1 {
		return path, fmt.Errorf("the resource name %s is not unique", resource)
	}
	return path, fmt.Errorf("there is no resource %s", resource)
}

func convertSources(sources map[string]resourceMetadata) map[string]string {
	list := make(map[string]string)
	for k, v := range sources {
		if len(v.subResources) > 0 {
			for sub, subWithPathName := range convertSources(v.subResources) {
				if subWithPathName != "" {
					withPathname := fmt.Sprintf("/%s:*%s", k, subWithPathName)
					if v.pathName != "" {
						withPathname = fmt.Sprintf("/%s:{%s}%s", k, v.pathName, subWithPathName)
					}
					list[fmt.Sprintf("/%s%s", k, sub)] = withPathname
				}
			}
		}
		withPathname := fmt.Sprintf("/%s:*/", k)
		if v.pathName != "" {
			withPathname = fmt.Sprintf("/%s:{%s}/", k, v.pathName)
		}
		list[fmt.Sprintf("/%s/", k)] = withPathname
	}
	return list
}

// registerResourceAction register resource actions
func registerResourceAction(resource string, actions ...string) {
	lock.Lock()
	defer lock.Unlock()
	if resourceActions == nil {
		resourceActions = make(map[string][]string)
	}
	path, err := checkResourcePath(resource)
	if err != nil {
		panic(fmt.Sprintf("resource %s is not exist", resource))
	}
	resource = path
	if _, exist := resourceActions[resource]; exist {
		for _, action := range actions {
			if !utils.StringsContain(resourceActions[resource], action) {
				resourceActions[resource] = append(resourceActions[resource], action)
			}
		}
	} else {
		resourceActions[resource] = actions
	}
}

type rbacServiceImpl struct {
	Store      datastore.DataStore `inject:"datastore"`
	KubeClient client.Client       `inject:"kubeClient"`
}

// RBACService implement RBAC-related business logic.
type RBACService interface {
	CheckPerm(resource string, actions ...string) func(req *restful.Request, res *restful.Response, chain *restful.FilterChain)
	GetUserPermissions(ctx context.Context, user *model.User, projectName string, withPlatform bool) ([]*model.Permission, error)
	CreateRole(ctx context.Context, projectName string, req apisv1.CreateRoleRequest) (*apisv1.RoleBase, error)
	DeleteRole(ctx context.Context, projectName, roleName string) error
	UpdateRole(ctx context.Context, projectName, roleName string, req apisv1.UpdateRoleRequest) (*apisv1.RoleBase, error)
	ListRole(ctx context.Context, projectName string, page, pageSize int) (*apisv1.ListRolesResponse, error)
	ListPermissionTemplate(ctx context.Context, projectName string) ([]apisv1.PermissionTemplateBase, error)
	ListPermissions(ctx context.Context, projectName string) ([]apisv1.PermissionBase, error)
	CreatePermission(ctx context.Context, projectName string, req apisv1.CreatePermissionRequest) (*apisv1.PermissionBase, error)
	DeletePermission(ctx context.Context, projectName, permName string) error
	SyncDefaultRoleAndUsersForProject(ctx context.Context, project *model.Project) error
	Init(ctx context.Context) error
}

// NewRBACService is the service service of RBAC
func NewRBACService() RBACService {
	rbacService := &rbacServiceImpl{}
	return rbacService
}

func (p *rbacServiceImpl) Init(ctx context.Context) error {
	count, _ := p.Store.Count(ctx, &model.Permission{}, &datastore.FilterOptions{
		IsNotExist: []datastore.IsNotExistQueryOption{
			{
				Key: "project",
			},
		},
	})
	if count == 0 {
		var batchData []datastore.Entity
		for _, policy := range defaultPlatformPermission {
			batchData = append(batchData, &model.Permission{
				Name:      policy.Name,
				Alias:     policy.Alias,
				Resources: policy.Resources,
				Actions:   policy.Actions,
				Effect:    policy.Effect,
			})
		}
		batchData = append(batchData, &model.Role{
			Name:        "admin",
			Alias:       "Admin",
			Permissions: []string{"admin"},
		})
		if err := p.Store.BatchAdd(ctx, batchData); err != nil {
			return fmt.Errorf("init the platform perm policies failure %w", err)
		}
	}

	if err := managePrivilegesForAdminUser(ctx, p.KubeClient, "admin", false); err != nil {
		return fmt.Errorf("failed to init the RBAC in cluster for the admin role %w", err)
	}
	return nil
}

// GetUserPermissions get user permission policies, if projectName is empty, will only get the platform permission policies
func (p *rbacServiceImpl) GetUserPermissions(ctx context.Context, user *model.User, projectName string, withPlatform bool) ([]*model.Permission, error) {
	var permissionNames []string
	var perms []*model.Permission
	if withPlatform && len(user.UserRoles) > 0 {
		entities, err := p.Store.List(ctx, &model.Role{}, &datastore.ListOptions{FilterOptions: datastore.FilterOptions{
			In: []datastore.InQueryOption{
				{
					Key:    "name",
					Values: user.UserRoles,
				},
			},
			IsNotExist: []datastore.IsNotExistQueryOption{
				{
					Key: "project",
				},
			},
		}})
		if err != nil {
			return nil, err
		}
		for _, entity := range entities {
			permissionNames = append(permissionNames, entity.(*model.Role).Permissions...)
		}
		perms, err = p.listPermPolices(ctx, "", permissionNames)
		if err != nil {
			return nil, err
		}
	}
	if projectName != "" {
		var projectUser = model.ProjectUser{
			ProjectName: projectName,
			Username:    user.Name,
		}
		var roles []string
		if err := p.Store.Get(ctx, &projectUser); err == nil {
			roles = append(roles, projectUser.UserRoles...)
		}
		if len(roles) > 0 {
			entities, err := p.Store.List(ctx, &model.Role{Project: projectName}, &datastore.ListOptions{FilterOptions: datastore.FilterOptions{In: []datastore.InQueryOption{
				{
					Key:    "name",
					Values: roles,
				},
			}}})
			if err != nil {
				return nil, err
			}
			for _, entity := range entities {
				permissionNames = append(permissionNames, entity.(*model.Role).Permissions...)
			}
			projectPerms, err := p.listPermPolices(ctx, projectName, permissionNames)
			if err != nil {
				return nil, err
			}
			perms = append(perms, projectPerms...)
		}
	}
	// with the default permissions
	perms = append(perms, &model.Permission{
		Name:      "cloudshell",
		Resources: []string{"cloudshell"},
		Actions:   []string{"*"},
		Effect:    "Allow",
	})
	return perms, nil
}

func (p *rbacServiceImpl) UpdatePermission(ctx context.Context, projectName string, permissionName string, req *apisv1.UpdatePermissionRequest) (*apisv1.PermissionBase, error) {
	perm := &model.Permission{
		Project: projectName,
		Name:    permissionName,
	}
	err := p.Store.Get(ctx, perm)
	if err != nil {
		if errors.Is(err, datastore.ErrRecordNotExist) {
			return nil, bcode.ErrPermissionNotExist
		}
	}
	//TODO: check req validate
	perm.Actions = req.Actions
	perm.Alias = req.Alias
	perm.Resources = req.Resources
	perm.Effect = req.Effect
	if err := p.Store.Put(ctx, perm); err != nil {
		return nil, err
	}
	return &apisv1.PermissionBase{
		Name:       perm.Name,
		Alias:      perm.Alias,
		Resources:  perm.Resources,
		Actions:    perm.Actions,
		Effect:     perm.Effect,
		CreateTime: perm.CreateTime,
		UpdateTime: perm.UpdateTime,
	}, nil
}

func (p *rbacServiceImpl) listPermPolices(ctx context.Context, projectName string, permissionNames []string) ([]*model.Permission, error) {
	if len(permissionNames) == 0 {
		return []*model.Permission{}, nil
	}
	filter := datastore.FilterOptions{In: []datastore.InQueryOption{
		{
			Key:    "name",
			Values: permissionNames,
		},
	}}
	if projectName == "" {
		filter.IsNotExist = append(filter.IsNotExist, datastore.IsNotExistQueryOption{
			Key: "project",
		})
	}
	permEntities, err := p.Store.List(ctx, &model.Permission{Project: projectName}, &datastore.ListOptions{FilterOptions: filter})
	if err != nil {
		return nil, err
	}
	var perms []*model.Permission
	for _, entity := range permEntities {
		perms = append(perms, entity.(*model.Permission))
	}
	return perms, nil
}

func (p *rbacServiceImpl) CheckPerm(resource string, actions ...string) func(req *restful.Request, res *restful.Response, chain *restful.FilterChain) {
	registerResourceAction(resource, actions...)
	f := func(req *restful.Request, res *restful.Response, chain *restful.FilterChain) {
		// get login user info
		userName, ok := req.Request.Context().Value(&apisv1.CtxKeyUser).(string)
		if !ok {
			bcode.ReturnError(req, res, bcode.ErrUnauthorized)
			return
		}
		user := &model.User{Name: userName}
		if err := p.Store.Get(req.Request.Context(), user); err != nil {
			bcode.ReturnError(req, res, bcode.ErrUnauthorized)
			return
		}
		path, err := checkResourcePath(resource)
		if err != nil {
			klog.Errorf("check resource path failure %s", err.Error())
			bcode.ReturnError(req, res, bcode.ErrForbidden)
			return
		}

		// multiple method for get the project name.
		getProjectName := func() string {
			if value := req.PathParameter("projectName"); value != "" {
				return value
			}
			if value := req.QueryParameter("project"); value != "" {
				return value
			}
			if value := req.QueryParameter("projectName"); value != "" {
				return value
			}
			if appName := req.PathParameter(ResourceMaps["project"].subResources["application"].pathName); appName != "" {
				app := &model.Application{Name: appName}
				if err := p.Store.Get(req.Request.Context(), app); err == nil {
					return app.Project
				}
			}
			if envName := req.PathParameter(ResourceMaps["project"].subResources["environment"].pathName); envName != "" {
				env := &model.Env{Name: envName}
				if err := p.Store.Get(req.Request.Context(), env); err == nil {
					return env.Project
				}
			}
			return ""
		}

		ra := &RequestResourceAction{}
		ra.SetResourceWithName(path, func(name string) string {
			if name == ResourceMaps["project"].pathName {
				return getProjectName()
			}
			return req.PathParameter(name)
		})
		ra.SetActions(actions)

		// get user's perm list.
		projectName := getProjectName()
		permissions, err := p.GetUserPermissions(req.Request.Context(), user, projectName, true)
		if err != nil {
			klog.Errorf("get user's perm policies failure %s, user is %s", err.Error(), user.Name)
			bcode.ReturnError(req, res, bcode.ErrForbidden)
			return
		}
		if !ra.Match(permissions) {
			bcode.ReturnError(req, res, bcode.ErrForbidden)
			return
		}
		apiserverutils.SetUsernameAndProjectInRequestContext(req, userName, projectName)
		chain.ProcessFilter(req, res)
	}
	return f
}

func (p *rbacServiceImpl) CreateRole(ctx context.Context, projectName string, req apisv1.CreateRoleRequest) (*apisv1.RoleBase, error) {
	if projectName != "" {
		var project = model.Project{
			Name: projectName,
		}
		if err := p.Store.Get(ctx, &project); err != nil {
			return nil, bcode.ErrProjectIsNotExist
		}
	}
	if len(req.Permissions) == 0 {
		return nil, bcode.ErrRolePermissionCheckFailure
	}
	policies, err := p.listPermPolices(ctx, projectName, req.Permissions)
	if err != nil || len(policies) != len(req.Permissions) {
		return nil, bcode.ErrRolePermissionCheckFailure
	}
	var role = model.Role{
		Name:        req.Name,
		Alias:       req.Alias,
		Project:     projectName,
		Permissions: req.Permissions,
	}
	if err := p.Store.Add(ctx, &role); err != nil {
		if errors.Is(err, datastore.ErrRecordExist) {
			return nil, bcode.ErrRoleIsExist
		}
		return nil, err
	}
	return assembler.ConvertRole2DTO(&role, policies), nil
}

func (p *rbacServiceImpl) DeleteRole(ctx context.Context, projectName, roleName string) error {
	var role = model.Role{
		Name:    roleName,
		Project: projectName,
	}
	if err := p.Store.Delete(ctx, &role); err != nil {
		if errors.Is(err, datastore.ErrRecordNotExist) {
			return bcode.ErrRoleIsNotExist
		}
		return err
	}
	return nil
}

func (p *rbacServiceImpl) DeletePermission(ctx context.Context, projectName, permName string) error {
	roles, _, err := repository.ListRoles(ctx, p.Store, projectName, 0, 0)
	if err != nil {
		klog.Errorf("fail to list the roles: %s", err.Error())
		return bcode.ErrPermissionIsUsed
	}
	for _, role := range roles {
		for _, p := range role.Permissions {
			if p == permName {
				return bcode.ErrPermissionIsUsed
			}
		}
	}

	var perm = model.Permission{
		Name:    permName,
		Project: projectName,
	}
	if err := p.Store.Delete(ctx, &perm); err != nil {
		if errors.Is(err, datastore.ErrRecordNotExist) {
			return bcode.ErrRoleIsNotExist
		}
		return err
	}
	return nil
}

func (p *rbacServiceImpl) UpdateRole(ctx context.Context, projectName, roleName string, req apisv1.UpdateRoleRequest) (*apisv1.RoleBase, error) {
	if projectName != "" {
		var project = model.Project{
			Name: projectName,
		}
		if err := p.Store.Get(ctx, &project); err != nil {
			return nil, bcode.ErrProjectIsNotExist
		}
	}
	if len(req.Permissions) == 0 {
		return nil, bcode.ErrRolePermissionCheckFailure
	}
	policies, err := p.listPermPolices(ctx, projectName, req.Permissions)
	if err != nil || len(policies) != len(req.Permissions) {
		return nil, bcode.ErrRolePermissionCheckFailure
	}
	var role = model.Role{
		Name:    roleName,
		Project: projectName,
	}
	if err := p.Store.Get(ctx, &role); err != nil {
		if errors.Is(err, datastore.ErrRecordNotExist) {
			return nil, bcode.ErrRoleIsNotExist
		}
		return nil, err
	}
	role.Alias = req.Alias
	role.Permissions = req.Permissions
	if err := p.Store.Put(ctx, &role); err != nil {
		return nil, err
	}
	return assembler.ConvertRole2DTO(&role, policies), nil
}

func (p *rbacServiceImpl) ListRole(ctx context.Context, projectName string, page, pageSize int) (*apisv1.ListRolesResponse, error) {
	roles, count, err := repository.ListRoles(ctx, p.Store, projectName, 0, 0)
	if err != nil {
		return nil, err
	}
	var policySet = make(map[string]string)
	for _, role := range roles {
		for _, p := range role.Permissions {
			policySet[p] = p
		}
	}

	policies, err := p.listPermPolices(ctx, projectName, utils.MapKey2Array(policySet))
	if err != nil {
		klog.Errorf("list perm policies failure %s", err.Error())
	}
	var policyMap = make(map[string]*model.Permission)
	for i, policy := range policies {
		policyMap[policy.Name] = policies[i]
	}
	var res apisv1.ListRolesResponse
	for _, role := range roles {
		var rolePolicies []*model.Permission
		for _, perm := range role.Permissions {
			rolePolicies = append(rolePolicies, policyMap[perm])
		}
		res.Roles = append(res.Roles, assembler.ConvertRole2DTO(role, rolePolicies))
	}
	res.Total = count
	return &res, nil
}

// ListPermissionTemplate TODO:
func (p *rbacServiceImpl) ListPermissionTemplate(ctx context.Context, projectName string) ([]apisv1.PermissionTemplateBase, error) {
	return nil, nil
}

func (p *rbacServiceImpl) ListPermissions(ctx context.Context, projectName string) ([]apisv1.PermissionBase, error) {
	var filter datastore.FilterOptions
	if projectName == "" {
		filter.IsNotExist = append(filter.IsNotExist, datastore.IsNotExistQueryOption{
			Key: "project",
		})
	}
	permEntities, err := p.Store.List(ctx, &model.Permission{Project: projectName}, &datastore.ListOptions{FilterOptions: filter})
	if err != nil {
		return nil, err
	}
	var perms []apisv1.PermissionBase
	for _, entity := range permEntities {
		perm := entity.(*model.Permission)
		perms = append(perms, apisv1.PermissionBase{
			Name:       perm.Name,
			Alias:      perm.Alias,
			Resources:  perm.Resources,
			Actions:    perm.Actions,
			Effect:     perm.Effect,
			CreateTime: perm.CreateTime,
			UpdateTime: perm.UpdateTime,
		})
	}
	return perms, nil
}

func (p *rbacServiceImpl) CreatePermission(ctx context.Context, projectName string, req apisv1.CreatePermissionRequest) (*apisv1.PermissionBase, error) {
	if projectName != "" {
		var project = model.Project{
			Name: projectName,
		}
		if err := p.Store.Get(ctx, &project); err != nil {
			return nil, bcode.ErrProjectIsNotExist
		}
	}
	if len(req.Resources) == 0 {
		return nil, bcode.ErrRolePermissionCheckFailure
	}

	if len(req.Actions) == 0 {
		req.Actions = []string{"*"}
	}

	if req.Effect == "" {
		req.Effect = "Allow"
	}

	var permission = model.Permission{
		Name:      req.Name,
		Alias:     req.Alias,
		Project:   projectName,
		Resources: req.Resources,
		Actions:   req.Actions,
		Effect:    req.Effect,
	}

	if err := p.Store.Add(ctx, &permission); err != nil {
		if errors.Is(err, datastore.ErrRecordExist) {
			return nil, bcode.ErrPermissionIsExist
		}
		return nil, err
	}
	return assembler.ConvertPermission2DTO(&permission), nil
}

func (p *rbacServiceImpl) SyncDefaultRoleAndUsersForProject(ctx context.Context, project *model.Project) error {

	permissions, err := p.ListPermissions(ctx, project.Name)
	if err != nil {
		return err
	}
	var permissionMap = map[string]apisv1.PermissionBase{}
	for i, per := range permissions {
		permissionMap[per.Name] = permissions[i]
	}

	var batchData []datastore.Entity
	for _, permissionTemp := range defaultProjectPermissionTemplate {
		var rra = RequestResourceAction{}
		var formattedResource []string
		for _, resource := range permissionTemp.Resources {
			rra.SetResourceWithName(resource, func(name string) string {
				if name == ResourceMaps["project"].pathName {
					return project.Name
				}
				return ""
			})
			formattedResource = append(formattedResource, rra.GetResource().String())
		}
		permission := &model.Permission{
			Name:      permissionTemp.Name,
			Alias:     permissionTemp.Alias,
			Project:   project.Name,
			Resources: formattedResource,
			Actions:   permissionTemp.Actions,
			Effect:    permissionTemp.Effect,
		}
		if perm, exist := permissionMap[permissionTemp.Name]; exist {
			if !utils.EqualSlice(perm.Resources, permissionTemp.Resources) || utils.EqualSlice(perm.Actions, permissionTemp.Actions) {
				if err := p.Store.Put(ctx, permission); err != nil {
					return err
				}
			}
			continue
		}
		batchData = append(batchData, permission)
	}

	if len(permissions) == 0 {
		batchData = append(batchData, &model.Role{
			Name:        "app-developer",
			Alias:       "App Developer",
			Permissions: []string{"project-view", "app-management", "env-management", "config-management", "pipeline-management"},
			Project:     project.Name,
		}, &model.Role{
			Name:        "project-admin",
			Alias:       "Project Admin",
			Permissions: []string{"project-view", "app-management", "env-management", "pipeline-management", "config-management", "role-management"},
			Project:     project.Name,
		}, &model.Role{
			Name:        "project-viewer",
			Alias:       "Project Viewer",
			Permissions: []string{"project-view"},
			Project:     project.Name,
		})
		if project.Owner != "" {
			var projectUser = &model.ProjectUser{
				ProjectName: project.Name,
				UserRoles:   []string{"project-admin"},
				Username:    project.Owner,
			}
			batchData = append(batchData, projectUser)
		}
	}

	return p.Store.BatchAdd(ctx, batchData)
}

// ResourceName it is similar to ARNs
// <type>:<value>/<type>:<value>
type ResourceName struct {
	Type  string
	Value string
	Next  *ResourceName
}

// ParseResourceName parse string to ResourceName
func ParseResourceName(resource string) *ResourceName {
	RNs := strings.Split(resource, "/")
	var resourceName = ResourceName{}
	var current = &resourceName
	for _, rn := range RNs {
		rnData := strings.Split(rn, ":")
		if len(rnData) == 2 {
			current.Type = rnData[0]
			current.Value = rnData[1]
		}
		if len(rnData) == 1 {
			current.Type = rnData[0]
			current.Value = "*"
		}
		var next = &ResourceName{}
		current.Next = next
		current = next
	}
	return &resourceName
}

// Match the resource types same with target and resource value include
// target resource means request resources
func (r *ResourceName) Match(target *ResourceName) bool {
	current := r
	currentTarget := target
	for current != nil && current.Type != "" {
		if current.Type == "*" {
			return true
		}
		if currentTarget == nil || currentTarget.Type == "" {
			return false
		}
		if current.Type != currentTarget.Type {
			return false
		}
		if current.Value != currentTarget.Value && current.Value != "*" {
			return false
		}
		current = current.Next
		currentTarget = currentTarget.Next
	}
	if currentTarget != nil && currentTarget.Type != "" {
		return false
	}
	return true
}

func (r *ResourceName) String() string {
	strBuilder := &strings.Builder{}
	current := r
	for current != nil && current.Type != "" {
		strBuilder.WriteString(fmt.Sprintf("%s:%s/", current.Type, current.Value))
		current = current.Next
	}
	return strings.TrimSuffix(strBuilder.String(), "/")
}

// RequestResourceAction resource permission boundary
type RequestResourceAction struct {
	resource *ResourceName
	actions  []string
}

// SetResourceWithName format resource and assign a value from path parameter
func (r *RequestResourceAction) SetResourceWithName(resource string, pathParameter func(name string) string) {
	resultKey := reg.FindAllString(resource, -1)
	for _, sourcekey := range resultKey {
		key := sourcekey[1 : len(sourcekey)-1]
		value := pathParameter(key)
		if value == "" {
			value = "*"
		}
		resource = strings.Replace(resource, sourcekey, value, 1)
	}
	r.resource = ParseResourceName(resource)
}

// GetResource return the resource after be formated
func (r *RequestResourceAction) GetResource() *ResourceName {
	return r.resource
}

// SetActions set request actions
func (r *RequestResourceAction) SetActions(actions []string) {
	r.actions = actions
}

func (r *RequestResourceAction) match(policy *model.Permission) bool {
	// match actions, the policy actions will include the actions of request
	if !utils.SliceIncludeSlice(policy.Actions, r.actions) && !utils.StringsContain(policy.Actions, "*") {
		return false
	}
	// match resources
	for _, resource := range policy.Resources {
		resourceName := ParseResourceName(resource)
		if resourceName.Match(r.resource) {
			return true
		}
	}
	return false
}

// Match determines whether the request resources and actions matches the user permission set.
func (r *RequestResourceAction) Match(policies []*model.Permission) bool {
	for _, policy := range policies {
		if strings.EqualFold(policy.Effect, "deny") {
			if r.match(policy) {
				return false
			}
		}
	}
	for _, policy := range policies {
		if strings.EqualFold(policy.Effect, "allow") || policy.Effect == "" {
			if r.match(policy) {
				return true
			}
		}
	}
	return false
}

// managePrivilegesForAdminUser grant or revoke privileges for admin user
func managePrivilegesForAdminUser(ctx context.Context, cli client.Client, roleName string, revoke bool) error {
	p := &auth.ScopedPrivilege{Cluster: types.ClusterLocalName}
	identity := &auth.Identity{Groups: []string{apiserverutils.KubeVelaAdminGroupPrefix + roleName}}
	writer := &bytes.Buffer{}
	f, msg := auth.GrantPrivileges, "GrantPrivileges"
	if revoke {
		f, msg = auth.RevokePrivileges, "RevokePrivileges"
	}
	if err := f(ctx, cli, []auth.PrivilegeDescription{p}, identity, writer); err != nil {
		return err
	}
	klog.Infof("%s: %s", msg, writer.String())
	return nil
}
