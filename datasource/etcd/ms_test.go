/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package etcd

import (
	"github.com/apache/servicecomb-service-center/datasource"
	"github.com/apache/servicecomb-service-center/pkg/log"
	pb "github.com/apache/servicecomb-service-center/pkg/registry"
	"github.com/apache/servicecomb-service-center/pkg/util"
	"github.com/apache/servicecomb-service-center/server/core"
	"github.com/apache/servicecomb-service-center/server/core/proto"
	"github.com/apache/servicecomb-service-center/server/plugin/quota"
	scerr "github.com/apache/servicecomb-service-center/server/scerror"
	"github.com/apache/servicecomb-service-center/server/service"
	"github.com/go-chassis/go-archaius"
	"github.com/stretchr/testify/assert"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestInit(t *testing.T) {
	_ = archaius.Init(archaius.WithMemorySource())
	_ = archaius.Set("servicecomb.datasource.name", "etcd")
}

func TestService_Register(t *testing.T) {
	t.Run("Register service after init & install, should pass", func(t *testing.T) {
		datasource.Install("etcd", func(opts datasource.Options) (datasource.DataSource, error) {
			return NewDataSource(opts), nil
		})

		err := datasource.Init(datasource.Options{
			Endpoint:       "",
			PluginImplName: datasource.ImplName(archaius.GetString("servicecomb.datasource.name", "etcd")),
		})
		assert.NoError(t, err)

		size := quota.DefaultSchemaQuota + 1
		paths := make([]*pb.ServicePath, 0, size)
		properties := make(map[string]string, size)
		for i := 0; i < size; i++ {
			s := strconv.Itoa(i) + strings.Repeat("x", 253)
			paths = append(paths, &pb.ServicePath{Path: s, Property: map[string]string{s: s}})
			properties[s] = s
		}
		request := &pb.CreateServiceRequest{
			Service: &pb.MicroService{
				AppId:       "service-ms-appID",
				ServiceName: "service-ms-serviceName",
				Version:     "32767.32767.32767.32767",
				Alias:       "service-ms-alias",
				Level:       "BACK",
				Status:      "UP",
				Schemas:     []string{"service-ms-schema"},
				Paths:       paths,
				Properties:  properties,
				Framework: &pb.FrameWorkProperty{
					Name:    "service-ms-frameworkName",
					Version: "service-ms-frameworkVersion",
				},
				RegisterBy: "SDK",
				Timestamp:  strconv.FormatInt(time.Now().Unix(), 10),
			},
		}
		request.Service.ModTimestamp = request.Service.Timestamp
		resp, err := datasource.Instance().RegisterService(getContext(), request)
		assert.NotNil(t, resp)
		assert.NoError(t, err)
		assert.Equal(t, proto.Response_SUCCESS, resp.Response.GetCode())
	})

	t.Run("register service with same key", func(t *testing.T) {
		datasource.Install("etcd", func(opts datasource.Options) (datasource.DataSource, error) {
			return NewDataSource(opts), nil
		})
		err := datasource.Init(datasource.Options{
			Endpoint:       "",
			PluginImplName: datasource.ImplName(archaius.GetString("servicecomb.datasource.name", "etcd")),
		})
		assert.NoError(t, err)

		// serviceName: some-relay-ms-service-name
		// alias: sr-ms-service-name
		resp, err := datasource.Instance().RegisterService(getContext(), &pb.CreateServiceRequest{
			Service: &pb.MicroService{
				ServiceName: "some-relay-ms-service-name",
				Alias:       "sr-ms-service-name",
				AppId:       "default",
				Version:     "1.0.0",
				Level:       "FRONT",
				Schemas: []string{
					"xxxxxxxx",
				},
				Status: "UP",
			},
		})
		assert.NotNil(t, resp)
		assert.NoError(t, err)
		assert.Equal(t, proto.Response_SUCCESS, resp.Response.GetCode())
		sameId := resp.ServiceId

		// serviceName: some-relay-ms-service-name
		// alias: sr1-ms-service-name
		resp, err = datasource.Instance().RegisterService(getContext(), &pb.CreateServiceRequest{
			Service: &pb.MicroService{
				ServiceName: "some-relay-ms-service-name",
				Alias:       "sr1-ms-service-name",
				AppId:       "default",
				Version:     "1.0.0",
				Level:       "FRONT",
				Schemas: []string{
					"xxxxxxxx",
				},
				Status: "UP",
			},
		})
		assert.NotNil(t, resp)
		assert.NoError(t, err)
		assert.Equal(t, scerr.ErrServiceAlreadyExists, resp.Response.GetCode())

		// serviceName: some-relay1-ms-service-name
		// alias: sr-ms-service-name
		resp, err = datasource.Instance().RegisterService(getContext(), &pb.CreateServiceRequest{
			Service: &pb.MicroService{
				ServiceName: "some-relay1-ms-service-name",
				Alias:       "sr-ms-service-name",
				AppId:       "default",
				Version:     "1.0.0",
				Level:       "FRONT",
				Schemas: []string{
					"xxxxxxxx",
				},
				Status: "UP",
			},
		})
		assert.NotNil(t, resp)
		assert.NoError(t, err)
		assert.Equal(t, scerr.ErrServiceAlreadyExists, resp.Response.GetCode())

		// serviceName: some-relay1-ms-service-name
		// alias: sr-ms-service-name
		// add serviceId field: sameId
		resp, err = datasource.Instance().RegisterService(getContext(), &pb.CreateServiceRequest{
			Service: &pb.MicroService{
				ServiceId:   sameId,
				ServiceName: "some-relay1-ms-service-name",
				Alias:       "sr-ms-service-name",
				AppId:       "default",
				Version:     "1.0.0",
				Level:       "FRONT",
				Schemas: []string{
					"xxxxxxxx",
				},
				Status: "UP",
			},
		})
		assert.NotNil(t, resp)
		assert.NoError(t, err)
		assert.Equal(t, proto.Response_SUCCESS, resp.Response.GetCode())

		// serviceName: some-relay-ms-service-name
		// alias: sr1-ms-service-name
		// serviceId: sameId
		resp, err = datasource.Instance().RegisterService(getContext(), &pb.CreateServiceRequest{
			Service: &pb.MicroService{
				ServiceId:   sameId,
				ServiceName: "some-relay-ms-service-name",
				Alias:       "sr1-ms-service-name",
				AppId:       "default",
				Version:     "1.0.0",
				Level:       "FRONT",
				Schemas: []string{
					"xxxxxxxx",
				},
				Status: "UP",
			},
		})
		assert.NotNil(t, resp)
		assert.NoError(t, err)
		assert.Equal(t, proto.Response_SUCCESS, resp.Response.GetCode())

		// serviceName: some-relay-ms-service-name
		// alias: sr1-ms-service-name
		// serviceId: custom-id-ms-service-id -- different
		resp, err = datasource.Instance().RegisterService(getContext(), &pb.CreateServiceRequest{
			Service: &pb.MicroService{
				ServiceId:   "custom-id-ms-service-id",
				ServiceName: "some-relay-ms-service-name",
				Alias:       "sr1-ms-service-name",
				AppId:       "default",
				Version:     "1.0.0",
				Level:       "FRONT",
				Schemas: []string{
					"xxxxxxxx",
				},
				Status: "UP",
			},
		})
		assert.NotNil(t, resp)
		assert.NoError(t, err)
		assert.Equal(t, scerr.ErrServiceAlreadyExists, resp.Response.GetCode())

		// serviceName: some-relay1-ms-service-name
		// alias: sr-ms-service-name
		// serviceId: custom-id-ms-service-id -- different
		resp, err = datasource.Instance().RegisterService(getContext(), &pb.CreateServiceRequest{
			Service: &pb.MicroService{
				ServiceId:   "custom-id-ms-service-id",
				ServiceName: "some-relay1-ms-service-name",
				Alias:       "sr-ms-service-name",
				AppId:       "default",
				Version:     "1.0.0",
				Level:       "FRONT",
				Schemas: []string{
					"xxxxxxxx",
				},
				Status: "UP",
			},
		})
		assert.NotNil(t, resp)
		assert.NoError(t, err)
		assert.Equal(t, scerr.ErrServiceAlreadyExists, resp.Response.GetCode())
	})

	t.Run("same serviceId,different service, can not register again,error is same as the service register twice",
		func(t *testing.T) {
			datasource.Install("etcd", func(opts datasource.Options) (datasource.DataSource, error) {
				return NewDataSource(opts), nil
			})
			err := datasource.Init(datasource.Options{
				Endpoint:       "",
				PluginImplName: datasource.ImplName(archaius.GetString("servicecomb.datasource.name", "etcd")),
			})
			assert.NoError(t, err)

			resp, err := datasource.Instance().RegisterService(getContext(), &pb.CreateServiceRequest{
				Service: &pb.MicroService{
					ServiceId:   "same-serviceId-service-ms",
					ServiceName: "serviceA-service-ms",
					AppId:       "default-service-ms",
					Version:     "1.0.0",
					Level:       "FRONT",
					Schemas: []string{
						"xxxxxxxx",
					},
					Status: "UP",
				},
			})

			assert.NotNil(t, resp)
			assert.NoError(t, err)
			assert.Equal(t, resp.Response.GetCode(), proto.Response_SUCCESS)

			// same serviceId with different service name
			resp, err = datasource.Instance().RegisterService(getContext(), &pb.CreateServiceRequest{
				Service: &pb.MicroService{
					ServiceId:   "same-serviceId-service-ms",
					ServiceName: "serviceB-service-ms",
					AppId:       "default-service-ms",
					Version:     "1.0.0",
					Level:       "FRONT",
					Schemas: []string{
						"xxxxxxxx",
					},
					Status: "UP",
				},
			})
			assert.NotNil(t, resp)
			assert.NoError(t, err)
			assert.Equal(t, scerr.ErrServiceAlreadyExists, resp.Response.GetCode())
		})
}

func TestService_Get(t *testing.T) {
	// get service test
	t.Run("query all services, should pass", func(t *testing.T) {
		datasource.Install("etcd", func(opts datasource.Options) (datasource.DataSource, error) {
			return NewDataSource(opts), nil
		})
		err := datasource.Init(datasource.Options{
			Endpoint:       "",
			PluginImplName: datasource.ImplName(archaius.GetString("servicecomb.datasource.name", "etcd")),
		})
		assert.NoError(t, err)
		resp, err := datasource.Instance().GetServices(getContext(), &pb.GetServicesRequest{})
		assert.NoError(t, err)
		assert.Greater(t, len(resp.Services), 0)
	})

	t.Run("get a exist service, should pass", func(t *testing.T) {
		datasource.Install("etcd", func(opts datasource.Options) (datasource.DataSource, error) {
			return NewDataSource(opts), nil
		})
		err := datasource.Init(datasource.Options{
			Endpoint:       "",
			PluginImplName: datasource.ImplName(archaius.GetString("servicecomb.datasource.name", "etcd")),
		})
		assert.NoError(t, err)

		request := &pb.CreateServiceRequest{
			Service: &pb.MicroService{
				ServiceId:   "ms-service-query-id",
				ServiceName: "ms-service-query",
				AppId:       "default",
				Version:     "1.0.4",
				Level:       "BACK",
				Properties:  make(map[string]string),
			},
		}

		resp, err := datasource.Instance().RegisterService(getContext(), request)
		assert.NoError(t, err)
		assert.Equal(t, resp.Response.GetCode(), proto.Response_SUCCESS)

		// search service by serviceID
		queryResp, err := datasource.Instance().GetService(getContext(), &pb.GetServiceRequest{
			ServiceId: "ms-service-query-id",
		})
		assert.NoError(t, err)
		assert.Equal(t, queryResp.Response.GetCode(), proto.Response_SUCCESS)
	})

	t.Run("query a service by a not existed serviceId, should not pass", func(t *testing.T) {
		datasource.Install("etcd", func(opts datasource.Options) (datasource.DataSource, error) {
			return NewDataSource(opts), nil
		})
		err := datasource.Init(datasource.Options{
			Endpoint:       "",
			PluginImplName: datasource.ImplName(archaius.GetString("servicecomb.datasource.name", "etcd")),
		})
		assert.NoError(t, err)

		// not exist service
		resp, err := datasource.Instance().GetService(getContext(), &pb.GetServiceRequest{
			ServiceId: "no-exist-service",
		})
		assert.NoError(t, err)
		assert.Equal(t, resp.Response.GetCode(), scerr.ErrServiceNotExists)
	})
}

func TestService_Exist(t *testing.T) {
	var (
		serviceId1 string
		serviceId2 string
	)

	datasource.Install("etcd", func(opts datasource.Options) (datasource.DataSource, error) {
		return NewDataSource(opts), nil
	})

	err := datasource.Init(datasource.Options{
		Endpoint:       "",
		PluginImplName: datasource.ImplName(archaius.GetString("servicecomb.datasource.name", "etcd")),
	})
	assert.NoError(t, err)

	t.Run("create service", func(t *testing.T) {
		svc := &pb.MicroService{
			Alias:       "es_service_ms",
			ServiceName: "exist_service_service_ms",
			AppId:       "exist_appId_service_ms",
			Version:     "1.0.0",
			Level:       "FRONT",
			Schemas: []string{
				"first_schemaId_service_ms",
			},
			Status: "UP",
		}
		resp, err := datasource.Instance().RegisterService(getContext(), &pb.CreateServiceRequest{
			Service: svc,
		})
		assert.NoError(t, err)
		assert.NotEqual(t, "", resp.ServiceId)
		serviceId1 = resp.ServiceId

		svc.ServiceId = ""
		svc.Environment = pb.ENV_PROD
		resp, err = datasource.Instance().RegisterService(getContext(), &pb.CreateServiceRequest{
			Service: svc,
		})
		assert.NoError(t, err)
		assert.NotEqual(t, "", resp.ServiceId)
		serviceId2 = resp.ServiceId
	})

	t.Run("check exist when service does not exist", func(t *testing.T) {
		log.Info("check by querying a not exist serviceName")
		resp, err := datasource.Instance().ExistService(getContext(), &pb.GetExistenceRequest{
			Type:        service.ExistTypeMicroservice,
			AppId:       "exist_appId",
			ServiceName: "notExistService_service_ms",
			Version:     "1.0.0",
		})
		assert.NoError(t, err)
		assert.Equal(t, scerr.ErrServiceNotExists, resp.Response.GetCode())

		log.Info("check by querying a not exist env")
		resp, err = datasource.Instance().ExistService(getContext(), &pb.GetExistenceRequest{
			Type:        service.ExistTypeMicroservice,
			Environment: pb.ENV_TEST,
			AppId:       "exist_appId_service_ms",
			ServiceName: "exist_service_service_ms",
			Version:     "1.0.0",
		})
		assert.NoError(t, err)
		assert.Equal(t, scerr.ErrServiceNotExists, resp.Response.GetCode())

		log.Info("check by querying a not exist env with alias")
		resp, err = datasource.Instance().ExistService(getContext(), &pb.GetExistenceRequest{
			Type:        service.ExistTypeMicroservice,
			Environment: pb.ENV_TEST,
			AppId:       "exist_appId_service_ms",
			ServiceName: "es_service_ms",
			Version:     "1.0.0",
		})
		assert.NoError(t, err)
		assert.Equal(t, scerr.ErrServiceNotExists, resp.Response.GetCode())

		log.Info("check by querying with a mismatching version")
		resp, err = datasource.Instance().ExistService(getContext(), &pb.GetExistenceRequest{
			Type:        service.ExistTypeMicroservice,
			AppId:       "exist_appId_service_ms",
			ServiceName: "exist_service_service_ms",
			Version:     "2.0.0",
		})
		assert.NoError(t, err)
		assert.Equal(t, scerr.ErrServiceNotExists, resp.Response.GetCode())
		resp, err = datasource.Instance().ExistService(getContext(), &pb.GetExistenceRequest{
			Type:        service.ExistTypeMicroservice,
			AppId:       "exist_appId_service_ms",
			ServiceName: "exist_service_service_ms",
			Version:     "0.0.0-1.0.0",
		})
		assert.NoError(t, err)
		assert.Equal(t, scerr.ErrServiceVersionNotExists, resp.Response.GetCode())
	})

	t.Run("check exist when service exists", func(t *testing.T) {
		log.Info("search with serviceName")
		resp, err := datasource.Instance().ExistService(getContext(), &pb.GetExistenceRequest{
			Type:        service.ExistTypeMicroservice,
			AppId:       "exist_appId_service_ms",
			ServiceName: "exist_service_service_ms",
			Version:     "1.0.0",
		})
		assert.NoError(t, err)
		assert.Equal(t, serviceId1, resp.ServiceId)

		log.Info("check with serviceName and env")
		resp, err = datasource.Instance().ExistService(getContext(), &pb.GetExistenceRequest{
			Type:        service.ExistTypeMicroservice,
			Environment: pb.ENV_PROD,
			AppId:       "exist_appId_service_ms",
			ServiceName: "exist_service_service_ms",
			Version:     "1.0.0",
		})
		assert.NoError(t, err)
		assert.Equal(t, serviceId2, resp.ServiceId)

		log.Info("check with alias")
		resp, err = datasource.Instance().ExistService(getContext(), &pb.GetExistenceRequest{
			Type:        service.ExistTypeMicroservice,
			AppId:       "exist_appId_service_ms",
			ServiceName: "es_service_ms",
			Version:     "1.0.0",
		})
		assert.NoError(t, err)
		assert.Equal(t, serviceId1, resp.ServiceId)

		log.Info("check with alias and env")
		resp, err = datasource.Instance().ExistService(getContext(), &pb.GetExistenceRequest{
			Type:        service.ExistTypeMicroservice,
			Environment: pb.ENV_PROD,
			AppId:       "exist_appId_service_ms",
			ServiceName: "es_service_ms",
			Version:     "1.0.0",
		})
		assert.NoError(t, err)
		assert.Equal(t, serviceId2, resp.ServiceId)

		log.Info("check with latest versionRule")
		resp, err = datasource.Instance().ExistService(getContext(), &pb.GetExistenceRequest{
			Type:        service.ExistTypeMicroservice,
			AppId:       "exist_appId_service_ms",
			ServiceName: "es_service_ms",
			Version:     "latest",
		})
		assert.NoError(t, err)
		assert.Equal(t, serviceId1, resp.ServiceId)

		log.Info("check with 1.0.0+ versionRule")
		resp, err = datasource.Instance().ExistService(getContext(), &pb.GetExistenceRequest{
			Type:        service.ExistTypeMicroservice,
			AppId:       "exist_appId_service_ms",
			ServiceName: "es_service_ms",
			Version:     "1.0.0+",
		})
		assert.NoError(t, err)
		assert.Equal(t, serviceId1, resp.ServiceId)

		log.Info("check with range versionRule")
		resp, err = datasource.Instance().ExistService(getContext(), &pb.GetExistenceRequest{
			Type:        service.ExistTypeMicroservice,
			AppId:       "exist_appId_service_ms",
			ServiceName: "es_service_ms",
			Version:     "0.9.1-1.0.1",
		})
		assert.NoError(t, err)
		assert.Equal(t, serviceId1, resp.ServiceId)
	})
}

func TestService_Update(t *testing.T) {
	datasource.Install("etcd", func(opts datasource.Options) (datasource.DataSource, error) {
		return NewDataSource(opts), nil
	})

	err := datasource.Init(datasource.Options{
		Endpoint:       "",
		PluginImplName: datasource.ImplName(archaius.GetString("servicecomb.datasource.name", "etcd")),
	})
	assert.NoError(t, err)

	var serviceId string

	t.Run("create service", func(t *testing.T) {
		resp, err := datasource.Instance().RegisterService(getContext(), &pb.CreateServiceRequest{
			Service: &pb.MicroService{
				Alias:       "es_service_ms",
				ServiceName: "update_prop_service_service_ms",
				AppId:       "update_prop_appId_service_ms",
				Version:     "1.0.0",
				Level:       "FRONT",
				Status:      "UP",
			},
		})
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, proto.Response_SUCCESS, resp.Response.GetCode())
		assert.NotEqual(t, "", resp.ServiceId)
		serviceId = resp.ServiceId
	})

	t.Run("update properties while properties not nil", func(t *testing.T) {
		log.Info("shuold pass")
		request := &pb.UpdateServicePropsRequest{
			ServiceId:  serviceId,
			Properties: make(map[string]string),
		}
		request2 := &pb.UpdateServicePropsRequest{
			ServiceId:  serviceId,
			Properties: make(map[string]string),
		}
		request.Properties["test"] = "1"
		request2.Properties["k"] = "v"
		resp, err := datasource.Instance().UpdateService(getContext(), request)
		assert.NoError(t, err)
		assert.Equal(t, proto.Response_SUCCESS, resp.Response.GetCode())

		resp, err = datasource.Instance().UpdateService(getContext(), request2)
		assert.NoError(t, err)
		assert.Equal(t, proto.Response_SUCCESS, resp.Response.GetCode())

		respGetService, err := datasource.Instance().GetService(getContext(), &pb.GetServiceRequest{
			ServiceId: serviceId,
		})
		assert.NoError(t, err)
		assert.Equal(t, serviceId, respGetService.Service.ServiceId)
		assert.Equal(t, "", respGetService.Service.Properties["test"])
		assert.Equal(t, "v", respGetService.Service.Properties["k"])
	})

	t.Run("update service that does not exist", func(t *testing.T) {
		log.Info("it should be failed")
		r := &pb.UpdateServicePropsRequest{
			ServiceId:  "not_exist_service_service_ms",
			Properties: make(map[string]string),
		}
		resp, err := datasource.Instance().UpdateService(getContext(), r)
		assert.NoError(t, err)
		assert.NotEqual(t, proto.Response_SUCCESS, resp.Response.GetCode())
	})

	t.Run("update service by removing the properties", func(t *testing.T) {
		log.Info("it should pass")
		r := &pb.UpdateServicePropsRequest{
			ServiceId:  serviceId,
			Properties: nil,
		}
		resp, err := datasource.Instance().UpdateService(getContext(), r)
		assert.NoError(t, err)
		assert.Equal(t, proto.Response_SUCCESS, resp.Response.GetCode())

		log.Info("remove properties for service with empty serviceId")
		r = &pb.UpdateServicePropsRequest{
			ServiceId:  "",
			Properties: map[string]string{},
		}
		resp, err = datasource.Instance().UpdateService(getContext(), r)
		assert.NoError(t, err)
		assert.NotEqual(t, proto.Response_SUCCESS, resp.Response.GetCode())
	})
}

func TestService_Delete(t *testing.T) {
	datasource.Install("etcd", func(opts datasource.Options) (datasource.DataSource, error) {
		return NewDataSource(opts), nil
	})

	err := datasource.Init(datasource.Options{
		Endpoint:       "",
		PluginImplName: datasource.ImplName(archaius.GetString("servicecomb.datasource.name", "etcd")),
	})
	assert.NoError(t, err)

	var (
		serviceContainInstId string
		serviceNoInstId      string
	)

	t.Run("create service & instance", func(t *testing.T) {
		respCreate, err := datasource.Instance().RegisterService(getContext(), &pb.CreateServiceRequest{
			Service: &pb.MicroService{
				ServiceName: "delete_service_with_inst_ms",
				AppId:       "delete_service_ms",
				Version:     "1.0.0",
				Level:       "FRONT",
				Status:      "UP",
			},
		})
		assert.NoError(t, err)
		assert.Equal(t, proto.Response_SUCCESS, respCreate.Response.GetCode())
		serviceContainInstId = respCreate.ServiceId

		log.Info("attach instance")
		instance := &pb.MicroServiceInstance{
			ServiceId: serviceContainInstId,
			Endpoints: []string{
				"deleteService:127.0.0.1:8080",
			},
			HostName: "delete-host-ms",
			Status:   pb.MSI_UP,
		}
		respCreateIns, err := datasource.Instance().RegisterInstance(getContext(), &pb.RegisterInstanceRequest{
			Instance: instance,
		})
		assert.NoError(t, err)
		assert.Equal(t, proto.Response_SUCCESS, respCreateIns.Response.GetCode())

		log.Info("create service without instance")
		provider := &pb.MicroService{
			ServiceName: "delete_service_no_inst_ms",
			AppId:       "delete_service_ms",
			Version:     "1.0.0",
			Level:       "FRONT",
			Status:      "UP",
		}
		respCreate, err = datasource.Instance().RegisterService(getContext(), &pb.CreateServiceRequest{
			Service: provider,
		})
		assert.NoError(t, err)
		assert.Equal(t, proto.Response_SUCCESS, respCreate.Response.GetCode())
		serviceNoInstId = respCreate.ServiceId

		respCreate, err = datasource.Instance().RegisterService(getContext(), &pb.CreateServiceRequest{
			Service: &pb.MicroService{
				ServiceName: "delete_service_consumer_ms",
				AppId:       "delete_service_ms",
				Version:     "1.0.0",
				Level:       "FRONT",
				Status:      "UP",
			},
		})
		assert.NoError(t, err)
		assert.Equal(t, proto.Response_SUCCESS, respCreate.Response.GetCode())
	})

	t.Run("delete a service which contains instances with no force flag", func(t *testing.T) {
		log.Info("should not pass")
		resp, err := datasource.Instance().UnregisterService(getContext(), &pb.DeleteServiceRequest{
			ServiceId: serviceContainInstId,
			Force:     false,
		})
		assert.NoError(t, err)
		assert.NotEqual(t, proto.Response_SUCCESS, resp.Response.GetCode())
	})

	t.Run("delete a service which contains instances with force flag", func(t *testing.T) {
		log.Info("should pass")
		resp, err := datasource.Instance().UnregisterService(getContext(), &pb.DeleteServiceRequest{
			ServiceId: serviceContainInstId,
			Force:     true,
		})
		assert.NoError(t, err)
		assert.Equal(t, proto.Response_SUCCESS, resp.Response.GetCode())
	})

	// todo: add delete service depended by consumer after finishing dependency management

	t.Run("delete a service which depended by consumer with force flag", func(t *testing.T) {
		log.Info("should pass")
		resp, err := datasource.Instance().UnregisterService(getContext(), &pb.DeleteServiceRequest{
			ServiceId: serviceNoInstId,
			Force:     true,
		})
		assert.NoError(t, err)
		assert.Equal(t, proto.Response_SUCCESS, resp.Response.GetCode())
	})

	t.Run("delete a service with no force flag", func(t *testing.T) {
		log.Info("should not pass")
		resp, err := datasource.Instance().UnregisterService(getContext(), &pb.DeleteServiceRequest{
			ServiceId: serviceNoInstId,
			Force:     false,
		})
		assert.NoError(t, err)
		assert.NotEqual(t, proto.Response_SUCCESS, resp.Response.GetCode())
	})
}

func TestInstance_Create(t *testing.T) {
	datasource.Install("etcd", func(opts datasource.Options) (datasource.DataSource, error) {
		return NewDataSource(opts), nil
	})

	err := datasource.Init(datasource.Options{
		Endpoint:       "",
		PluginImplName: datasource.ImplName(archaius.GetString("servicecomb.datasource.name", "etcd")),
	})
	assert.NoError(t, err)

	var serviceId string

	t.Run("create service", func(t *testing.T) {
		respCreateService, err := datasource.Instance().RegisterService(getContext(), &pb.CreateServiceRequest{
			Service: &pb.MicroService{
				ServiceName: "create_instance_service_ms",
				AppId:       "create_instance_ms",
				Version:     "1.0.0",
				Level:       "FRONT",
				Status:      pb.MS_UP,
			},
		})

		assert.NoError(t, err)
		assert.Equal(t, proto.Response_SUCCESS, respCreateService.Response.GetCode())
		serviceId = respCreateService.ServiceId
	})

	t.Run("register instance", func(t *testing.T) {
		respCreateInst, err := datasource.Instance().RegisterInstance(getContext(), &pb.RegisterInstanceRequest{
			Instance: &pb.MicroServiceInstance{
				ServiceId: serviceId,
				Endpoints: []string{
					"createInstance_ms:127.0.0.1:8080",
				},
				HostName: "UT-HOST",
				Status:   pb.MSI_UP,
			},
		})
		assert.NoError(t, err)
		assert.Equal(t, proto.Response_SUCCESS, respCreateInst.Response.GetCode())
		assert.NotEqual(t, "", respCreateInst.InstanceId)

		respCreateInst, err = datasource.Instance().RegisterInstance(getContext(), &pb.RegisterInstanceRequest{
			Instance: &pb.MicroServiceInstance{
				InstanceId: "customId_ms",
				ServiceId:  serviceId,
				Endpoints: []string{
					"createInstance_ms:127.0.0.1:8080",
				},
				HostName: "UT-HOST",
				Status:   pb.MSI_UP,
			},
		})
		assert.NoError(t, err)
		assert.Equal(t, proto.Response_SUCCESS, respCreateInst.Response.GetCode())
		assert.Equal(t, "customId_ms", respCreateInst.InstanceId)
	})

	t.Run("update the same instance", func(t *testing.T) {
		instance := &pb.MicroServiceInstance{
			ServiceId: serviceId,
			Endpoints: []string{
				"sameInstance:127.0.0.1:8080",
			},
			HostName: "UT-HOST",
			Status:   pb.MSI_UP,
		}
		resp, err := datasource.Instance().RegisterInstance(getContext(), &pb.RegisterInstanceRequest{
			Instance: instance,
		})
		assert.NoError(t, err)
		assert.Equal(t, proto.Response_SUCCESS, resp.Response.GetCode())

		resp, err = datasource.Instance().RegisterInstance(getContext(), &pb.RegisterInstanceRequest{
			Instance: instance,
		})
		assert.NoError(t, err)
		assert.Equal(t, proto.Response_SUCCESS, resp.Response.GetCode())
		assert.Equal(t, instance.InstanceId, resp.InstanceId)
	})
}

func TestInstance_HeartBeat(t *testing.T) {
	datasource.Install("etcd", func(opts datasource.Options) (datasource.DataSource, error) {
		return NewDataSource(opts), nil
	})
	err := datasource.Init(datasource.Options{
		Endpoint:       "",
		PluginImplName: datasource.ImplName(archaius.GetString("servicecomb.datasource.name", "etcd")),
	})
	assert.NoError(t, err)

	var (
		serviceId   string
		instanceId1 string
		instanceId2 string
	)

	t.Run("register service and instance, should pass", func(t *testing.T) {
		log.Info("register service")
		respCreateService, err := datasource.Instance().RegisterService(getContext(), &pb.CreateServiceRequest{
			Service: &pb.MicroService{
				ServiceName: "heartbeat_service_ms",
				AppId:       "heartbeat_service_ms",
				Version:     "1.0.0",
				Level:       "FRONT",
				Status:      pb.MS_UP,
			},
		})
		assert.NoError(t, err)
		assert.Equal(t, proto.Response_SUCCESS, respCreateService.Response.GetCode())
		serviceId = respCreateService.ServiceId

		respCreateInstance, err := datasource.Instance().RegisterInstance(getContext(), &pb.RegisterInstanceRequest{
			Instance: &pb.MicroServiceInstance{
				ServiceId: serviceId,
				HostName:  "UT-HOST-MS",
				Endpoints: []string{
					"heartbeat:127.0.0.1:8080",
				},
				Status: pb.MSI_UP,
			},
		})
		assert.NoError(t, err)
		assert.Equal(t, proto.Response_SUCCESS, respCreateInstance.Response.GetCode())
		instanceId1 = respCreateInstance.InstanceId

		respCreateInstance, err = datasource.Instance().RegisterInstance(getContext(), &pb.RegisterInstanceRequest{
			Instance: &pb.MicroServiceInstance{
				ServiceId: serviceId,
				HostName:  "UT-HOST-MS",
				Endpoints: []string{
					"heartbeat:127.0.0.2:8080",
				},
				Status: pb.MSI_UP,
			},
		})
		assert.NoError(t, err)
		assert.Equal(t, proto.Response_SUCCESS, respCreateInstance.Response.GetCode())
		instanceId2 = respCreateInstance.InstanceId
	})

	t.Run("update a lease", func(t *testing.T) {
		log.Info("valid instance")
		resp, err := datasource.Instance().Heartbeat(getContext(), &pb.HeartbeatRequest{
			ServiceId:  serviceId,
			InstanceId: instanceId1,
		})
		assert.NoError(t, err)
		assert.Equal(t, proto.Response_SUCCESS, resp.Response.GetCode())

		log.Info("serviceId does not exist")
		resp, err = datasource.Instance().Heartbeat(getContext(), &pb.HeartbeatRequest{
			ServiceId:  "100000000000",
			InstanceId: instanceId1,
		})
		assert.NoError(t, err)
		assert.NotEqual(t, proto.Response_SUCCESS, resp.Response.GetCode())

		log.Info("instance does not exist")
		resp, err = datasource.Instance().Heartbeat(getContext(), &pb.HeartbeatRequest{
			ServiceId:  serviceId,
			InstanceId: "not-exist-ins",
		})
		assert.NoError(t, err)
		assert.NotEqual(t, proto.Response_SUCCESS, resp.Response.GetCode())
	})

	t.Run("batch update lease", func(t *testing.T) {
		log.Info("request contains at least 1 instances")
		resp, err := datasource.Instance().HeartbeatSet(getContext(), &pb.HeartbeatSetRequest{
			Instances: []*pb.HeartbeatSetElement{
				{
					ServiceId:  serviceId,
					InstanceId: instanceId1,
				},
				{
					ServiceId:  serviceId,
					InstanceId: instanceId2,
				},
			},
		})
		assert.NoError(t, err)
		assert.Equal(t, proto.Response_SUCCESS, resp.Response.GetCode())
	})
}

func TestInstance_Update(t *testing.T) {
	datasource.Install("etcd", func(opts datasource.Options) (datasource.DataSource, error) {
		return NewDataSource(opts), nil
	})
	err := datasource.Init(datasource.Options{
		Endpoint:       "",
		PluginImplName: datasource.ImplName(archaius.GetString("servicecomb.datasource.name", "etcd")),
	})
	assert.NoError(t, err)

	var (
		serviceId  string
		instanceId string
	)

	t.Run("register service and instance, should pass", func(t *testing.T) {
		log.Info("register service")
		respCreateService, err := datasource.Instance().RegisterService(getContext(), &pb.CreateServiceRequest{
			Service: &pb.MicroService{
				ServiceName: "update_instance_service_ms",
				AppId:       "update_instance_service_ms",
				Version:     "1.0.0",
				Level:       "FRONT",
				Status:      pb.MS_UP,
			},
		})
		assert.NoError(t, err)
		assert.Equal(t, proto.Response_SUCCESS, respCreateService.Response.GetCode())
		serviceId = respCreateService.ServiceId

		log.Info("create instance")
		respCreateInstance, err := datasource.Instance().RegisterInstance(getContext(), &pb.RegisterInstanceRequest{
			Instance: &pb.MicroServiceInstance{
				ServiceId: serviceId,
				Endpoints: []string{
					"updateInstance:127.0.0.1:8080",
				},
				HostName:   "UT-HOST-MS",
				Status:     pb.MSI_UP,
				Properties: map[string]string{"nodeIP": "test"},
			},
		})
		assert.NoError(t, err)
		assert.Equal(t, proto.Response_SUCCESS, respCreateInstance.Response.GetCode())
		instanceId = respCreateInstance.InstanceId
	})

	t.Run("update instance status", func(t *testing.T) {
		log.Info("update instance status to DOWN")
		respUpdateStatus, err := datasource.Instance().UpdateInstanceStatus(getContext(), &pb.UpdateInstanceStatusRequest{
			ServiceId:  serviceId,
			InstanceId: instanceId,
			Status:     pb.MSI_DOWN,
		})
		assert.NoError(t, err)
		assert.Equal(t, proto.Response_SUCCESS, respUpdateStatus.Response.GetCode())

		log.Info("update instance status to OUTOFSERVICE")
		respUpdateStatus, err = datasource.Instance().UpdateInstanceStatus(getContext(), &pb.UpdateInstanceStatusRequest{
			ServiceId:  serviceId,
			InstanceId: instanceId,
			Status:     pb.MSI_OUTOFSERVICE,
		})
		assert.NoError(t, err)
		assert.Equal(t, proto.Response_SUCCESS, respUpdateStatus.Response.GetCode())

		log.Info("update instance status to STARTING")
		respUpdateStatus, err = datasource.Instance().UpdateInstanceStatus(getContext(), &pb.UpdateInstanceStatusRequest{
			ServiceId:  serviceId,
			InstanceId: instanceId,
			Status:     pb.MSI_STARTING,
		})
		assert.NoError(t, err)
		assert.Equal(t, proto.Response_SUCCESS, respUpdateStatus.Response.GetCode())

		log.Info("update instance status to TESTING")
		respUpdateStatus, err = datasource.Instance().UpdateInstanceStatus(getContext(), &pb.UpdateInstanceStatusRequest{
			ServiceId:  serviceId,
			InstanceId: instanceId,
			Status:     pb.MSI_TESTING,
		})
		assert.NoError(t, err)
		assert.Equal(t, proto.Response_SUCCESS, respUpdateStatus.Response.GetCode())

		log.Info("update instance status to UP")
		respUpdateStatus, err = datasource.Instance().UpdateInstanceStatus(getContext(), &pb.UpdateInstanceStatusRequest{
			ServiceId:  serviceId,
			InstanceId: instanceId,
			Status:     pb.MSI_UP,
		})
		assert.NoError(t, err)
		assert.Equal(t, proto.Response_SUCCESS, respUpdateStatus.Response.GetCode())

		log.Info("update instance status with a not exist instance")
		respUpdateStatus, err = datasource.Instance().UpdateInstanceStatus(getContext(), &pb.UpdateInstanceStatusRequest{
			ServiceId:  serviceId,
			InstanceId: "notexistins",
			Status:     pb.MSI_STARTING,
		})
		assert.NoError(t, err)
		assert.NotEqual(t, proto.Response_SUCCESS, respUpdateStatus.Response.GetCode())
	})

	t.Run("update instance properties", func(t *testing.T) {
		log.Info("update one properties")
		respUpdateProperties, err := datasource.Instance().UpdateInstanceProperties(getContext(),
			&pb.UpdateInstancePropsRequest{
				ServiceId:  serviceId,
				InstanceId: instanceId,
				Properties: map[string]string{
					"test": "test",
				},
			})
		assert.NoError(t, err)
		assert.Equal(t, proto.Response_SUCCESS, respUpdateProperties.Response.GetCode())

		log.Info("all max properties updated")
		size := 1000
		properties := make(map[string]string, size)
		for i := 0; i < size; i++ {
			s := strconv.Itoa(i) + strings.Repeat("x", 253)
			properties[s] = s
		}
		respUpdateProperties, err = datasource.Instance().UpdateInstanceProperties(getContext(),
			&pb.UpdateInstancePropsRequest{
				ServiceId:  serviceId,
				InstanceId: instanceId,
				Properties: properties,
			})
		assert.NoError(t, err)
		assert.Equal(t, proto.Response_SUCCESS, respUpdateProperties.Response.GetCode())

		log.Info("update instance that does not exist")
		respUpdateProperties, err = datasource.Instance().UpdateInstanceProperties(getContext(),
			&pb.UpdateInstancePropsRequest{
				ServiceId:  serviceId,
				InstanceId: "not_exist_ins",
				Properties: map[string]string{
					"test": "test",
				},
			})
		assert.NoError(t, err)
		assert.NotEqual(t, proto.Response_SUCCESS, respUpdateProperties.Response.GetCode())

		log.Info("remove properties")
		respUpdateProperties, err = datasource.Instance().UpdateInstanceProperties(getContext(),
			&pb.UpdateInstancePropsRequest{
				ServiceId:  serviceId,
				InstanceId: instanceId,
			})
		assert.NoError(t, err)
		assert.Equal(t, proto.Response_SUCCESS, respUpdateProperties.Response.GetCode())

		log.Info("update service that does not exist")
		respUpdateProperties, err = datasource.Instance().UpdateInstanceProperties(getContext(),
			&pb.UpdateInstancePropsRequest{
				ServiceId:  "not_exist_service",
				InstanceId: instanceId,
				Properties: map[string]string{
					"test": "test",
				},
			})
		assert.NoError(t, err)
		assert.NotEqual(t, proto.Response_SUCCESS, respUpdateProperties.Response.GetCode())
	})
}

func TestInstance_Query(t *testing.T) {
	datasource.Install("etcd", func(opts datasource.Options) (datasource.DataSource, error) {
		return NewDataSource(opts), nil
	})
	err := datasource.Init(datasource.Options{
		Endpoint:       "",
		PluginImplName: datasource.ImplName(archaius.GetString("servicecomb.datasource.name", "etcd")),
	})
	assert.NoError(t, err)

	var (
		serviceId1  string
		serviceId2  string
		serviceId3  string
		serviceId4  string
		serviceId5  string
		serviceId6  string
		serviceId7  string
		serviceId8  string
		serviceId9  string
		instanceId1 string
		instanceId2 string
		instanceId4 string
		instanceId5 string
		instanceId8 string
		instanceId9 string
	)

	t.Run("register services and instances for testInstance_query", func(t *testing.T) {
		respCreateService, err := datasource.Instance().RegisterService(getContext(), &pb.CreateServiceRequest{
			Service: &pb.MicroService{
				AppId:       "query_instance_ms",
				ServiceName: "query_instance_service_ms",
				Version:     "1.0.0",
				Level:       "FRONT",
				Status:      pb.MS_UP,
			},
		})
		assert.NoError(t, err)
		assert.Equal(t, proto.Response_SUCCESS, respCreateService.Response.GetCode())
		serviceId1 = respCreateService.ServiceId

		respCreateService, err = datasource.Instance().RegisterService(getContext(), &pb.CreateServiceRequest{
			Service: &pb.MicroService{
				AppId:       "query_instance_ms",
				ServiceName: "query_instance_service_ms",
				Version:     "1.0.5",
				Level:       "FRONT",
				Status:      pb.MS_UP,
			},
		})
		assert.NoError(t, err)
		assert.Equal(t, proto.Response_SUCCESS, respCreateService.Response.GetCode())
		serviceId2 = respCreateService.ServiceId

		respCreateService, err = datasource.Instance().RegisterService(getContext(), &pb.CreateServiceRequest{
			Service: &pb.MicroService{
				AppId:       "query_instance_diff_app_ms",
				ServiceName: "query_instance_service_ms",
				Version:     "1.0.0",
				Level:       "FRONT",
				Status:      pb.MS_UP,
			},
		})
		assert.NoError(t, err)
		assert.Equal(t, proto.Response_SUCCESS, respCreateService.Response.GetCode())
		serviceId3 = respCreateService.ServiceId

		respCreateService, err = datasource.Instance().RegisterService(getContext(), &pb.CreateServiceRequest{
			Service: &pb.MicroService{
				Environment: pb.ENV_PROD,
				AppId:       "query_instance_ms",
				ServiceName: "query_instance_diff_env_service_ms",
				Version:     "1.0.0",
				Level:       "FRONT",
				Status:      pb.MS_UP,
			},
		})
		assert.NoError(t, err)
		assert.Equal(t, proto.Response_SUCCESS, respCreateService.Response.GetCode())
		serviceId4 = respCreateService.ServiceId

		respCreateService, err = datasource.Instance().RegisterService(getContext(), &pb.CreateServiceRequest{
			Service: &pb.MicroService{
				Environment: pb.ENV_PROD,
				AppId:       "default",
				ServiceName: "query_instance_shared_provider_ms",
				Version:     "1.0.0",
				Level:       "FRONT",
				Status:      pb.MS_UP,
				Properties: map[string]string{
					proto.PROP_ALLOW_CROSS_APP: "true",
				},
			},
		})
		assert.NoError(t, err)
		assert.Equal(t, proto.Response_SUCCESS, respCreateService.Response.GetCode())
		serviceId5 = respCreateService.ServiceId

		respCreateService, err = datasource.Instance().RegisterService(
			util.SetDomainProject(util.CloneContext(getContext()), "user", "user"),
			&pb.CreateServiceRequest{
				Service: &pb.MicroService{
					AppId:       "default",
					ServiceName: "query_instance_diff_domain_consumer_ms",
					Version:     "1.0.0",
					Level:       "FRONT",
					Status:      pb.MS_UP,
				},
			})
		assert.NoError(t, err)
		assert.Equal(t, proto.Response_SUCCESS, respCreateService.Response.GetCode())
		serviceId6 = respCreateService.ServiceId

		respCreateService, err = datasource.Instance().RegisterService(getContext(), &pb.CreateServiceRequest{
			Service: &pb.MicroService{
				AppId:       "default",
				ServiceName: "query_instance_shared_consumer_ms",
				Version:     "1.0.0",
				Level:       "FRONT",
				Status:      pb.MS_UP,
			},
		})
		assert.NoError(t, err)
		assert.Equal(t, proto.Response_SUCCESS, respCreateService.Response.GetCode())
		serviceId7 = respCreateService.ServiceId

		respCreateService, err = datasource.Instance().RegisterService(getContext(), &pb.CreateServiceRequest{
			Service: &pb.MicroService{
				AppId:       "query_instance_ms",
				ServiceName: "query_instance_with_rev_ms",
				Version:     "1.0.0",
				Level:       "FRONT",
				Status:      pb.MS_UP,
			},
		})
		assert.NoError(t, err)
		assert.Equal(t, proto.Response_SUCCESS, respCreateService.Response.GetCode())
		serviceId8 = respCreateService.ServiceId

		respCreateService, err = datasource.Instance().RegisterService(getContext(), &pb.CreateServiceRequest{
			Service: &pb.MicroService{
				AppId:       "query_instance_ms",
				ServiceName: "batch_query_instance_with_rev_ms",
				Version:     "1.0.0",
				Level:       "FRONT",
				Status:      pb.MS_UP,
			},
		})
		assert.NoError(t, err)
		assert.Equal(t, proto.Response_SUCCESS, respCreateService.Response.GetCode())
		serviceId9 = respCreateService.ServiceId

		respCreateInstance, err := datasource.Instance().RegisterInstance(getContext(), &pb.RegisterInstanceRequest{
			Instance: &pb.MicroServiceInstance{
				ServiceId: serviceId1,
				HostName:  "UT-HOST-MS",
				Endpoints: []string{
					"find:127.0.0.1:8080",
				},
				Status: pb.MSI_UP,
			},
		})
		assert.NoError(t, err)
		assert.Equal(t, proto.Response_SUCCESS, respCreateInstance.Response.GetCode())
		instanceId1 = respCreateInstance.InstanceId

		respCreateInstance, err = datasource.Instance().RegisterInstance(getContext(), &pb.RegisterInstanceRequest{
			Instance: &pb.MicroServiceInstance{
				ServiceId: serviceId2,
				HostName:  "UT-HOST-MS",
				Endpoints: []string{
					"find:127.0.0.2:8080",
				},
				Status: pb.MSI_UP,
			},
		})
		assert.NoError(t, err)
		assert.Equal(t, proto.Response_SUCCESS, respCreateInstance.Response.GetCode())
		instanceId2 = respCreateInstance.InstanceId

		respCreateInstance, err = datasource.Instance().RegisterInstance(getContext(), &pb.RegisterInstanceRequest{
			Instance: &pb.MicroServiceInstance{
				ServiceId: serviceId4,
				HostName:  "UT-HOST-MS",
				Endpoints: []string{
					"find:127.0.0.4:8080",
				},
				Status: pb.MSI_UP,
			},
		})
		assert.NoError(t, err)
		assert.Equal(t, proto.Response_SUCCESS, respCreateInstance.Response.GetCode())
		instanceId4 = respCreateInstance.InstanceId

		respCreateInstance, err = datasource.Instance().RegisterInstance(getContext(), &pb.RegisterInstanceRequest{
			Instance: &pb.MicroServiceInstance{
				ServiceId: serviceId5,
				HostName:  "UT-HOST-MS",
				Endpoints: []string{
					"find:127.0.0.5:8080",
				},
				Status: pb.MSI_UP,
			},
		})
		assert.NoError(t, err)
		assert.Equal(t, proto.Response_SUCCESS, respCreateInstance.Response.GetCode())
		instanceId5 = respCreateInstance.InstanceId

		respCreateInstance, err = datasource.Instance().RegisterInstance(getContext(), &pb.RegisterInstanceRequest{
			Instance: &pb.MicroServiceInstance{
				ServiceId: serviceId8,
				HostName:  "UT-HOST-MS",
				Endpoints: []string{
					"find:127.0.0.8:8080",
				},
				Status: pb.MSI_UP,
			},
		})
		assert.NoError(t, err)
		assert.Equal(t, proto.Response_SUCCESS, respCreateInstance.Response.GetCode())
		instanceId8 = respCreateInstance.InstanceId

		respCreateInstance, err = datasource.Instance().RegisterInstance(getContext(), &pb.RegisterInstanceRequest{
			Instance: &pb.MicroServiceInstance{
				ServiceId: serviceId9,
				HostName:  "UT-HOST-MS",
				Endpoints: []string{
					"find:127.0.0.9:8080",
				},
				Status: pb.MSI_UP,
			},
		})
		assert.NoError(t, err)
		assert.Equal(t, proto.Response_SUCCESS, respCreateInstance.Response.GetCode())
		instanceId9 = respCreateInstance.InstanceId
	})

	t.Run("query instance", func(t *testing.T) {
		log.Info("find with version rule")
		respFind, err := datasource.Instance().FindInstances(getContext(), &pb.FindInstancesRequest{
			ConsumerServiceId: serviceId1,
			AppId:             "query_instance_ms",
			ServiceName:       "query_instance_service_ms",
			VersionRule:       "latest",
		})
		assert.NoError(t, err)
		assert.Equal(t, proto.Response_SUCCESS, respFind.Response.GetCode())
		assert.Equal(t, instanceId2, respFind.Instances[0].InstanceId)

		respFind, err = datasource.Instance().FindInstances(getContext(), &pb.FindInstancesRequest{
			ConsumerServiceId: serviceId1,
			AppId:             "query_instance_ms",
			ServiceName:       "query_instance_service_ms",
			VersionRule:       "1.0.0+",
			Tags:              []string{},
		})
		assert.NoError(t, err)
		assert.Equal(t, proto.Response_SUCCESS, respFind.Response.GetCode())
		assert.Equal(t, instanceId2, respFind.Instances[0].InstanceId)

		respFind, err = datasource.Instance().FindInstances(getContext(), &pb.FindInstancesRequest{
			ConsumerServiceId: serviceId1,
			AppId:             "query_instance_ms",
			ServiceName:       "query_instance_service_ms",
			VersionRule:       "1.0.0",
		})
		assert.NoError(t, err)
		assert.Equal(t, proto.Response_SUCCESS, respFind.Response.GetCode())
		assert.Equal(t, instanceId1, respFind.Instances[0].InstanceId)

		respFind, err = datasource.Instance().FindInstances(getContext(), &pb.FindInstancesRequest{
			ConsumerServiceId: serviceId1,
			AppId:             "query_instance",
			ServiceName:       "query_instance_service",
			VersionRule:       "0.0.0",
		})
		assert.NoError(t, err)
		assert.Equal(t, scerr.ErrServiceNotExists, respFind.Response.GetCode())

		log.Info("find with env")
		respFind, err = datasource.Instance().FindInstances(getContext(), &pb.FindInstancesRequest{
			ConsumerServiceId: serviceId4,
			AppId:             "query_instance_ms",
			ServiceName:       "query_instance_diff_env_service_ms",
			VersionRule:       "1.0.0",
		})
		assert.NoError(t, err)
		assert.Equal(t, proto.Response_SUCCESS, respFind.Response.GetCode())
		assert.Equal(t, 1, len(respFind.Instances))
		assert.Equal(t, instanceId4, respFind.Instances[0].InstanceId)

		respFind, err = datasource.Instance().FindInstances(getContext(), &pb.FindInstancesRequest{
			Environment: pb.ENV_PROD,
			AppId:       "query_instance_ms",
			ServiceName: "query_instance_diff_env_service_ms",
			VersionRule: "1.0.0",
		})
		assert.NoError(t, err)
		assert.Equal(t, proto.Response_SUCCESS, respFind.Response.GetCode())
		assert.Equal(t, 1, len(respFind.Instances))
		assert.Equal(t, instanceId4, respFind.Instances[0].InstanceId)

		log.Info("find with rev")
		ctx := util.SetContext(getContext(), util.CtxNocache, "")
		respFind, err = datasource.Instance().FindInstances(ctx, &pb.FindInstancesRequest{
			ConsumerServiceId: serviceId8,
			AppId:             "query_instance_ms",
			ServiceName:       "query_instance_with_rev_ms",
			VersionRule:       "1.0.0",
		})
		assert.NoError(t, err)
		assert.Equal(t, proto.Response_SUCCESS, respFind.Response.GetCode())
		rev, _ := ctx.Value(util.CtxResponseRevision).(string)
		assert.Equal(t, instanceId8, respFind.Instances[0].InstanceId)
		assert.NotEqual(t, 0, len(rev))

		util.SetContext(ctx, util.CtxRequestRevision, "x")
		respFind, err = datasource.Instance().FindInstances(ctx, &pb.FindInstancesRequest{
			ConsumerServiceId: serviceId8,
			AppId:             "query_instance_ms",
			ServiceName:       "query_instance_with_rev_ms",
			VersionRule:       "1.0.0",
		})
		assert.NoError(t, err)
		assert.Equal(t, proto.Response_SUCCESS, respFind.Response.GetCode())
		assert.Equal(t, instanceId8, respFind.Instances[0].InstanceId)
		assert.Equal(t, ctx.Value(util.CtxResponseRevision), rev)

		log.Info("find should return 200 if consumer is diff apps")
		respFind, err = datasource.Instance().FindInstances(getContext(), &pb.FindInstancesRequest{
			ConsumerServiceId: serviceId3,
			AppId:             "query_instance_ms",
			ServiceName:       "query_instance_service_ms",
			VersionRule:       "1.0.5",
		})
		assert.NoError(t, err)
		assert.Equal(t, proto.Response_SUCCESS, respFind.Response.GetCode())
		assert.Equal(t, 0, len(respFind.Instances))

		log.Info("provider tag does not exist")
		respFind, err = datasource.Instance().FindInstances(getContext(), &pb.FindInstancesRequest{
			ConsumerServiceId: serviceId1,
			AppId:             "query_instance_ms",
			ServiceName:       "query_instance_service_ms",
			VersionRule:       "latest",
			Tags:              []string{"not_exist_tag"},
		})
		assert.NoError(t, err)
		assert.Equal(t, proto.Response_SUCCESS, respFind.Response.GetCode())
		assert.Equal(t, 0, len(respFind.Instances))

		log.Info("shared service discovery")
		_ = os.Setenv("CSE_SHARED_SERVICES", "query_instance_shared_provider_ms")
		core.SetSharedMode()
		core.Service.Environment = pb.ENV_PROD
		respFind, err = datasource.Instance().FindInstances(
			util.SetTargetDomainProject(
				util.SetDomainProject(util.CloneContext(getContext()), "user", "user"),
				"default", "default"),
			&pb.FindInstancesRequest{
				ConsumerServiceId: serviceId6,
				AppId:             "default",
				ServiceName:       "query_instance_shared_provider_ms",
				VersionRule:       "1.0.0",
			})
		assert.NoError(t, err)
		assert.Equal(t, proto.Response_SUCCESS, respFind.Response.GetCode())
		assert.Equal(t, 1, len(respFind.Instances))
		assert.Equal(t, instanceId5, respFind.Instances[0].InstanceId)

		respFind, err = datasource.Instance().FindInstances(getContext(), &pb.FindInstancesRequest{
			ConsumerServiceId: serviceId7,
			AppId:             "default",
			ServiceName:       "query_instance_shared_provider_ms",
			VersionRule:       "1.0.0",
		})
		assert.NoError(t, err)
		assert.Equal(t, proto.Response_SUCCESS, respFind.Response.GetCode())
		assert.Equal(t, 1, len(respFind.Instances))
		assert.Equal(t, instanceId5, respFind.Instances[0].InstanceId)

		log.Info("query same domain deps")
		// todo finish ut after implementing GetConsumerDependencies interface

		core.Service.Environment = pb.ENV_DEV
	})

	t.Run("batch query instances", func(t *testing.T) {
		log.Info("find with version rule")
		respFind, err := datasource.Instance().BatchFind(getContext(), &pb.BatchFindInstancesRequest{
			ConsumerServiceId: serviceId1,
			Services: []*pb.FindService{
				{
					Service: &pb.MicroServiceKey{
						AppId:       "query_instance_ms",
						ServiceName: "query_instance_service_ms",
						Version:     "latest",
					},
				},
				{
					Service: &pb.MicroServiceKey{
						AppId:       "query_instance_ms",
						ServiceName: "query_instance_service_ms",
						Version:     "1.0.0+",
					},
				},
				{
					Service: &pb.MicroServiceKey{
						AppId:       "query_instance_ms",
						ServiceName: "query_instance_service_ms",
						Version:     "0.0.0",
					},
				},
			},
		})
		assert.NoError(t, err)
		assert.Equal(t, proto.Response_SUCCESS, respFind.Response.GetCode())
		assert.Equal(t, int64(0), respFind.Services.Updated[0].Index)
		assert.Equal(t, instanceId2, respFind.Services.Updated[0].Instances[0].InstanceId)
		assert.Equal(t, int64(1), respFind.Services.Updated[1].Index)
		assert.Equal(t, instanceId2, respFind.Services.Updated[1].Instances[0].InstanceId)
		assert.Equal(t, int64(2), respFind.Services.Failed[0].Indexes[0])
		assert.Equal(t, scerr.ErrServiceNotExists, respFind.Services.Failed[0].Error.Code)

		log.Info("find with env")
		respFind, err = datasource.Instance().BatchFind(getContext(), &pb.BatchFindInstancesRequest{
			ConsumerServiceId: serviceId4,
			Services: []*pb.FindService{
				{
					Service: &pb.MicroServiceKey{
						AppId:       "query_instance_ms",
						ServiceName: "query_instance_diff_env_service_ms",
						Version:     "1.0.0",
					},
				},
			},
		})
		assert.NoError(t, err)
		assert.Equal(t, proto.Response_SUCCESS, respFind.Response.GetCode())
		assert.Equal(t, 1, len(respFind.Services.Updated[0].Instances))
		assert.Equal(t, instanceId4, respFind.Services.Updated[0].Instances[0].InstanceId)

		log.Info("find with rev")
		ctx := util.SetContext(getContext(), util.CtxNocache, "")
		respFind, err = datasource.Instance().BatchFind(ctx, &pb.BatchFindInstancesRequest{
			ConsumerServiceId: serviceId8,
			Services: []*pb.FindService{
				{
					Service: &pb.MicroServiceKey{
						AppId:       "query_instance_ms",
						ServiceName: "query_instance_with_rev_ms",
						Version:     "1.0.0",
					},
				},
				{
					Service: &pb.MicroServiceKey{
						AppId:       "query_instance_ms",
						ServiceName: "batch_query_instance_with_rev_ms",
						Version:     "1.0.0",
					},
				},
			},
			Instances: []*pb.FindInstance{
				{
					Instance: &pb.HeartbeatSetElement{
						ServiceId:  serviceId9,
						InstanceId: instanceId9,
					},
				},
				{
					Instance: &pb.HeartbeatSetElement{
						ServiceId:  serviceId8,
						InstanceId: instanceId8,
					},
				},
			},
		})
		assert.NoError(t, err)
		assert.Equal(t, proto.Response_SUCCESS, respFind.Response.GetCode())
		rev := respFind.Services.Updated[0].Rev
		assert.Equal(t, int64(0), respFind.Services.Updated[0].Index)
		assert.Equal(t, int64(1), respFind.Services.Updated[1].Index)
		assert.Equal(t, instanceId8, respFind.Services.Updated[0].Instances[0].InstanceId)
		assert.Equal(t, instanceId9, respFind.Services.Updated[1].Instances[0].InstanceId)
		assert.NotEqual(t, 0, len(rev))
		instanceRev := respFind.Instances.Updated[0].Rev
		assert.Equal(t, int64(0), respFind.Instances.Updated[0].Index)
		assert.Equal(t, int64(1), respFind.Instances.Updated[1].Index)
		assert.Equal(t, instanceId9, respFind.Instances.Updated[0].Instances[0].InstanceId)
		assert.Equal(t, instanceId8, respFind.Instances.Updated[1].Instances[0].InstanceId)
		assert.NotEqual(t, 0, len(instanceRev))

		respFind, err = datasource.Instance().BatchFind(ctx, &pb.BatchFindInstancesRequest{
			ConsumerServiceId: serviceId8,
			Services: []*pb.FindService{
				{
					Service: &pb.MicroServiceKey{
						AppId:       "query_instance_ms",
						ServiceName: "query_instance_with_rev_ms",
						Version:     "1.0.0",
					},
					Rev: "x",
				},
			},
			Instances: []*pb.FindInstance{
				{
					Instance: &pb.HeartbeatSetElement{
						ServiceId:  serviceId9,
						InstanceId: instanceId9,
					},
					Rev: "x",
				},
			},
		})
		assert.NoError(t, err)
		assert.Equal(t, proto.Response_SUCCESS, respFind.Response.GetCode())
		assert.Equal(t, instanceId8, respFind.Services.Updated[0].Instances[0].InstanceId)
		assert.Equal(t, respFind.Services.Updated[0].Rev, rev)
		assert.Equal(t, instanceId9, respFind.Instances.Updated[0].Instances[0].InstanceId)
		assert.Equal(t, instanceRev, respFind.Instances.Updated[0].Rev)

		respFind, err = datasource.Instance().BatchFind(ctx, &pb.BatchFindInstancesRequest{
			ConsumerServiceId: serviceId8,
			Services: []*pb.FindService{
				{
					Service: &pb.MicroServiceKey{
						AppId:       "query_instance_ms",
						ServiceName: "query_instance_with_rev_ms",
						Version:     "1.0.0",
					},
					Rev: rev,
				},
			},
			Instances: []*pb.FindInstance{
				{
					Instance: &pb.HeartbeatSetElement{
						ServiceId:  serviceId9,
						InstanceId: instanceId9,
					},
					Rev: instanceRev,
				},
			},
		})
		assert.NoError(t, err)
		assert.Equal(t, proto.Response_SUCCESS, respFind.Response.GetCode())
		assert.Equal(t, int64(0), respFind.Services.NotModified[0])
		assert.Equal(t, int64(0), respFind.Instances.NotModified[0])

		log.Info("find should return 200 even if consumer is diff apps")
		respFind, err = datasource.Instance().BatchFind(getContext(), &pb.BatchFindInstancesRequest{
			ConsumerServiceId: serviceId3,
			Services: []*pb.FindService{
				{
					Service: &pb.MicroServiceKey{
						AppId:       "query_instance_ms",
						ServiceName: "query_instance_service_ms",
						Version:     "1.0.5",
					},
				},
			},
		})
		assert.NoError(t, err)
		assert.Equal(t, proto.Response_SUCCESS, respFind.Response.GetCode())
		assert.Equal(t, 0, len(respFind.Services.Updated[0].Instances))

		log.Info("shared service discovery")
		_ = os.Setenv("CSE_SHARED_SERVICES", "query_instance_shared_provider_ms")
		core.SetSharedMode()
		core.Service.Environment = pb.ENV_PROD
		respFind, err = datasource.Instance().BatchFind(
			util.SetTargetDomainProject(
				util.SetDomainProject(util.CloneContext(getContext()), "user", "user"),
				"default", "default"),
			&pb.BatchFindInstancesRequest{
				ConsumerServiceId: serviceId6,
				Services: []*pb.FindService{
					{
						Service: &pb.MicroServiceKey{
							AppId:       "default",
							ServiceName: "query_instance_shared_provider_ms",
							Version:     "1.0.0",
						},
					},
				},
			})
		assert.NoError(t, err)
		assert.Equal(t, proto.Response_SUCCESS, respFind.Response.GetCode())
		assert.Equal(t, 1, len(respFind.Services.Updated[0].Instances))
		assert.Equal(t, instanceId5, respFind.Services.Updated[0].Instances[0].InstanceId)

		respFind, err = datasource.Instance().BatchFind(getContext(), &pb.BatchFindInstancesRequest{
			ConsumerServiceId: serviceId7,
			Services: []*pb.FindService{
				{
					Service: &pb.MicroServiceKey{
						AppId:       "default",
						ServiceName: "query_instance_shared_provider_ms",
						Version:     "1.0.0",
					},
				},
			},
		})
		assert.NoError(t, err)
		assert.Equal(t, proto.Response_SUCCESS, respFind.Response.GetCode())
		assert.Equal(t, 1, len(respFind.Services.Updated[0].Instances))
		assert.Equal(t, instanceId5, respFind.Services.Updated[0].Instances[0].InstanceId)

		respFind, err = datasource.Instance().BatchFind(util.SetTargetDomainProject(
			util.SetDomainProject(util.CloneContext(getContext()), "user", "user"),
			"default", "default"),
			&pb.BatchFindInstancesRequest{
				ConsumerServiceId: serviceId6,
				Instances: []*pb.FindInstance{
					{
						Instance: &pb.HeartbeatSetElement{
							ServiceId:  serviceId5,
							InstanceId: instanceId5,
						},
					},
				},
			})
		assert.NoError(t, err)
		assert.Equal(t, proto.Response_SUCCESS, respFind.Response.GetCode())
		assert.Equal(t, scerr.ErrServiceNotExists, respFind.Instances.Failed[0].Error.Code)

		respFind, err = datasource.Instance().BatchFind(getContext(), &pb.BatchFindInstancesRequest{
			ConsumerServiceId: serviceId7,
			Instances: []*pb.FindInstance{
				{
					Instance: &pb.HeartbeatSetElement{
						ServiceId:  serviceId5,
						InstanceId: instanceId5,
					},
				},
			},
		})
		assert.NoError(t, err)
		assert.Equal(t, proto.Response_SUCCESS, respFind.Response.GetCode())
		assert.Equal(t, 1, len(respFind.Instances.Updated[0].Instances))
		assert.Equal(t, instanceId5, respFind.Instances.Updated[0].Instances[0].InstanceId)

		core.Service.Environment = pb.ENV_DEV
	})

	t.Run("query instances between diff dimensions", func(t *testing.T) {
		log.Info("diff appId")
		UTFunc := func(consumerId string, code int32) {
			respFind, err := datasource.Instance().GetInstances(getContext(), &pb.GetInstancesRequest{
				ConsumerServiceId: consumerId,
				ProviderServiceId: serviceId2,
			})
			assert.NoError(t, err)
			assert.Equal(t, code, respFind.Response.GetCode())
		}

		UTFunc(serviceId3, scerr.ErrServiceNotExists)

		UTFunc(serviceId1, proto.Response_SUCCESS)

		log.Info("diff env")
		respFind, err := datasource.Instance().GetInstances(getContext(), &pb.GetInstancesRequest{
			ConsumerServiceId: serviceId4,
			ProviderServiceId: serviceId2,
		})
		assert.NoError(t, err)
		assert.NotEqual(t, proto.Response_SUCCESS, respFind.Response.GetCode())
	})
}

func TestInstance_GetOne(t *testing.T) {
	datasource.Install("etcd", func(opts datasource.Options) (datasource.DataSource, error) {
		return NewDataSource(opts), nil
	})
	err := datasource.Init(datasource.Options{
		Endpoint:       "",
		PluginImplName: datasource.ImplName(archaius.GetString("servicecomb.datasource.name", "etcd")),
	})
	assert.NoError(t, err)

	var (
		serviceId1  string
		serviceId2  string
		serviceId3  string
		instanceId2 string
	)

	t.Run("register service and instances", func(t *testing.T) {
		respCreateService, err := datasource.Instance().RegisterService(getContext(), &pb.CreateServiceRequest{
			Service: &pb.MicroService{
				AppId:       "get_instance_ms",
				ServiceName: "get_instance_service_ms",
				Version:     "1.0.0",
				Level:       "FRONT",
				Status:      pb.MS_UP,
			},
		})
		assert.NoError(t, err)
		assert.Equal(t, proto.Response_SUCCESS, respCreateService.Response.GetCode())
		serviceId1 = respCreateService.ServiceId

		respCreateService, err = datasource.Instance().RegisterService(getContext(), &pb.CreateServiceRequest{
			Service: &pb.MicroService{
				AppId:       "get_instance_ms",
				ServiceName: "get_instance_service_ms",
				Version:     "1.0.5",
				Level:       "FRONT",
				Status:      pb.MS_UP,
			},
		})
		assert.NoError(t, err)
		assert.Equal(t, proto.Response_SUCCESS, respCreateService.Response.GetCode())
		serviceId2 = respCreateService.ServiceId

		respCreateInstance, err := datasource.Instance().RegisterInstance(getContext(), &pb.RegisterInstanceRequest{
			Instance: &pb.MicroServiceInstance{
				ServiceId: serviceId2,
				HostName:  "UT-HOST-MS",
				Endpoints: []string{
					"get:127.0.0.2:8080",
				},
				Status: pb.MSI_UP,
			},
		})
		assert.NoError(t, err)
		assert.Equal(t, proto.Response_SUCCESS, respCreateService.Response.GetCode())
		instanceId2 = respCreateInstance.InstanceId

		respCreateService, err = datasource.Instance().RegisterService(getContext(), &pb.CreateServiceRequest{
			Service: &pb.MicroService{
				AppId:       "get_instance_cross_ms",
				ServiceName: "get_instance_service_ms",
				Version:     "1.0.0",
				Level:       "FRONT",
				Status:      pb.MS_UP,
			},
		})
		assert.NoError(t, err)
		assert.Equal(t, proto.Response_SUCCESS, respCreateService.Response.GetCode())
		serviceId3 = respCreateService.ServiceId
	})

	t.Run("get one instance when invalid request", func(t *testing.T) {
		log.Info("find service itself")
		resp, err := datasource.Instance().GetInstance(getContext(), &pb.GetOneInstanceRequest{
			ConsumerServiceId:  serviceId2,
			ProviderServiceId:  serviceId2,
			ProviderInstanceId: instanceId2,
		})
		assert.NoError(t, err)
		assert.Equal(t, proto.Response_SUCCESS, resp.Response.GetCode())

		log.Info("consumer does not exist")
		resp, err = datasource.Instance().GetInstance(getContext(), &pb.GetOneInstanceRequest{
			ConsumerServiceId:  "not-exist-id-ms",
			ProviderServiceId:  serviceId2,
			ProviderInstanceId: instanceId2,
		})
		assert.NoError(t, err)
		assert.NotEqual(t, proto.Response_SUCCESS, resp.Response.GetCode())
	})

	t.Run("get between diff apps", func(t *testing.T) {
		resp, err := datasource.Instance().GetInstance(getContext(), &pb.GetOneInstanceRequest{
			ConsumerServiceId:  serviceId3,
			ProviderServiceId:  serviceId2,
			ProviderInstanceId: instanceId2,
		})
		assert.NoError(t, err)
		assert.Equal(t, scerr.ErrInstanceNotExists, resp.Response.GetCode())

		respAll, err := datasource.Instance().GetInstances(getContext(), &pb.GetInstancesRequest{
			ConsumerServiceId: serviceId3,
			ProviderServiceId: serviceId2,
		})
		assert.NoError(t, err)
		assert.NotEqual(t, proto.Response_SUCCESS, respAll.Response.GetCode())
	})

	t.Run("get instances when request is invalid", func(t *testing.T) {
		log.Info("consumer does not exist")
		resp, err := datasource.Instance().GetInstances(getContext(), &pb.GetInstancesRequest{
			ConsumerServiceId: "not-exist-service-ms",
			ProviderServiceId: serviceId2,
		})
		assert.NoError(t, err)
		assert.NotEqual(t, proto.Response_SUCCESS, resp.Response.GetCode())

		log.Info("consumer does not exist")
		resp, err = datasource.Instance().GetInstances(getContext(), &pb.GetInstancesRequest{
			ConsumerServiceId: serviceId1,
			ProviderServiceId: serviceId2,
		})
		assert.NoError(t, err)
		assert.Equal(t, proto.Response_SUCCESS, resp.Response.GetCode())
	})
}

func TestInstance_Unregister(t *testing.T) {
	datasource.Install("etcd", func(opts datasource.Options) (datasource.DataSource, error) {
		return NewDataSource(opts), nil
	})
	err := datasource.Init(datasource.Options{
		Endpoint:       "",
		PluginImplName: datasource.ImplName(archaius.GetString("servicecomb.datasource.name", "etcd")),
	})
	assert.NoError(t, err)

	var (
		serviceId  string
		instanceId string
	)

	t.Run("register service and instances", func(t *testing.T) {
		respCreateService, err := datasource.Instance().RegisterService(getContext(), &pb.CreateServiceRequest{
			Service: &pb.MicroService{
				AppId:       "unregister_instance_ms",
				ServiceName: "unregister_instance_service_ms",
				Version:     "1.0.5",
				Level:       "FRONT",
				Status:      pb.MS_UP,
			},
			Tags: map[string]string{
				"test": "test",
			},
		})
		assert.NoError(t, err)
		assert.Equal(t, proto.Response_SUCCESS, respCreateService.Response.GetCode())
		serviceId = respCreateService.ServiceId

		respCreateInstance, err := datasource.Instance().RegisterInstance(getContext(), &pb.RegisterInstanceRequest{
			Instance: &pb.MicroServiceInstance{
				ServiceId: serviceId,
				HostName:  "UT-HOST-MS",
				Endpoints: []string{
					"unregister:127.0.0.2:8080",
				},
				Status: pb.MSI_UP,
			},
		})
		assert.NoError(t, err)
		assert.Equal(t, proto.Response_SUCCESS, respCreateInstance.Response.GetCode())
		instanceId = respCreateInstance.InstanceId
	})

	t.Run("unregister instance", func(t *testing.T) {
		resp, err := datasource.Instance().UnregisterInstance(getContext(), &pb.UnregisterInstanceRequest{
			ServiceId:  serviceId,
			InstanceId: instanceId,
		})
		assert.NoError(t, err)
		assert.Equal(t, proto.Response_SUCCESS, resp.Response.GetCode())
	})

	t.Run("unregister instance when request is invalid", func(t *testing.T) {
		log.Info("service id does not exist")
		resp, err := datasource.Instance().UnregisterInstance(getContext(), &pb.UnregisterInstanceRequest{
			ServiceId:  "not-exist-id-ms",
			InstanceId: instanceId,
		})
		assert.NoError(t, err)
		assert.NotEqual(t, proto.Response_SUCCESS, resp.Response.GetCode())

		log.Info("instance id does not exist")
		resp, err = datasource.Instance().UnregisterInstance(getContext(), &pb.UnregisterInstanceRequest{
			ServiceId:  serviceId,
			InstanceId: "not-exist-id-ms",
		})
		assert.NoError(t, err)
		assert.NotEqual(t, proto.Response_SUCCESS, resp.Response.GetCode())
	})
}

func TestSchema_Create(t *testing.T) {
	datasource.Install("etcd", func(opts datasource.Options) (datasource.DataSource, error) {
		return NewDataSource(opts), nil
	})
	err := datasource.Init(datasource.Options{
		Endpoint:       "",
		PluginImplName: datasource.ImplName(archaius.GetString("servicecomb.datasource.name", "etcd")),
	})
	assert.NoError(t, err)

	var (
		serviceIdDev string
	)

	t.Run("create service, should pass", func(t *testing.T) {
		svc := &pb.MicroService{
			Alias:       "create_schema_group_service_ms",
			ServiceName: "create_schema_service_service_ms",
			Version:     "1.0.0",
			Level:       "FRONT",
			Status:      pb.MS_UP,
			Environment: pb.ENV_DEV,
		}
		resp, err := datasource.Instance().RegisterService(getContext(), &pb.CreateServiceRequest{
			Service: svc,
		})
		assert.NoError(t, err)
		assert.NotEqual(t, "", resp.ServiceId)
		assert.Equal(t, proto.Response_SUCCESS, resp.Response.GetCode())
		serviceIdDev = resp.ServiceId

		resp, err = datasource.Instance().RegisterService(getContext(), &pb.CreateServiceRequest{
			Service: &pb.MicroService{
				AppId:       "create_schema_group_service_ms",
				ServiceName: "create_schema_service_service_ms",
				Version:     "1.0.0",
				Level:       "FRONT",
				Status:      pb.MS_UP,
				Environment: pb.ENV_PROD,
			},
		})
		assert.NoError(t, err)
		assert.NotEqual(t, "", resp.ServiceId)
		assert.Equal(t, proto.Response_SUCCESS, resp.Response.GetCode())
	})

	t.Run("create schemas out of gauge", func(t *testing.T) {
		log.Info("create schemas out of gauge")
		size := quota.DefaultSchemaQuota + 1
		schemaIds := make([]string, 0, size)
		schemas := make([]*pb.Schema, 0, size)
		for i := 0; i < size; i++ {
			s := "ServiceCombTestTheLimitOfSchemasServiceMS" + strconv.Itoa(i)

			schemaIds = append(schemaIds, s)
			schemas = append(schemas, &pb.Schema{
				SchemaId: s,
				Schema:   s,
				Summary:  s,
			})
		}

		log.Info("batch modify schemas 1, should failed")
		resp, err := datasource.Instance().ModifySchemas(getContext(), &pb.ModifySchemasRequest{
			ServiceId: serviceIdDev,
			Schemas:   schemas,
		})
		assert.NoError(t, err)
		assert.Equal(t, scerr.ErrNotEnoughQuota, resp.Response.GetCode())

		log.Info("batch modify schemas 2")
		resp, err = datasource.Instance().ModifySchemas(getContext(), &pb.ModifySchemasRequest{
			ServiceId: serviceIdDev,
			Schemas:   schemas[:quota.DefaultSchemaQuota],
		})
		assert.NoError(t, err)
		assert.Equal(t, proto.Response_SUCCESS, resp.Response.GetCode())

		log.Info("should be failed in production env")
		resp, err = datasource.Instance().ModifySchemas(getContext(), &pb.ModifySchemasRequest{
			ServiceId: serviceIdDev,
			Schemas:   schemas,
		})
		assert.NoError(t, err)
		assert.Equal(t, scerr.ErrNotEnoughQuota, resp.Response.GetCode())
	})

	t.Run("batch create schemas in dev env", func(t *testing.T) {
		var (
			serviceIdDev1 string
			serviceIdDev2 string
		)

		log.Info("register service, should pass")
		resp, err := datasource.Instance().RegisterService(getContext(), &pb.CreateServiceRequest{
			Service: &pb.MicroService{
				AppId:       "create_schemas_dev_service_ms",
				ServiceName: "create_schemas_service_service_ms",
				Version:     "1.0.0",
				Level:       "FRONT",
				Status:      pb.MS_UP,
				Environment: pb.ENV_DEV,
			},
		})
		assert.NoError(t, err)
		assert.Equal(t, proto.Response_SUCCESS, resp.Response.GetCode())
		serviceIdDev1 = resp.ServiceId

		resp, err = datasource.Instance().RegisterService(getContext(), &pb.CreateServiceRequest{
			Service: &pb.MicroService{
				AppId:       "create_schemas_dev_service_ms",
				ServiceName: "create_schemas_service_service_ms",
				Version:     "1.0.1",
				Level:       "FRONT",
				Schemas: []string{
					"first_schemaId_service_ms",
				},
				Status:      pb.MS_UP,
				Environment: pb.ENV_DEV,
			},
		})
		assert.NoError(t, err)
		assert.Equal(t, proto.Response_SUCCESS, resp.Response.GetCode())
		serviceIdDev2 = resp.ServiceId

		log.Info("create schemas with service schemaId set is empty")
		schemas := []*pb.Schema{
			{
				SchemaId: "first_schemaId_service_ms",
				Schema:   "first_schema_service_ms",
				Summary:  "first0summary_service_ms",
			},
			{
				SchemaId: "first_schemaId_service_ms",
				Schema:   "first_schema_service_ms",
				Summary:  "first0summary_service_ms",
			},
		}
		respCreateSchema, err := datasource.Instance().ModifySchemas(getContext(), &pb.ModifySchemasRequest{
			ServiceId: serviceIdDev1,
			Schemas:   schemas,
		})
		assert.NoError(t, err)
		assert.Equal(t, proto.Response_SUCCESS, respCreateSchema.Response.GetCode())

		// todo: test GetAllSchemaInfo interface refers to schema_test line 342

		log.Info("modify schemas")
		schemas = []*pb.Schema{
			{
				SchemaId: "first_schemaId_service_ms",
				Schema:   "first_schema_change_service_ms",
				Summary:  "first0summary1change_service_ms",
			},
		}
		respCreateSchema, err = datasource.Instance().ModifySchemas(getContext(), &pb.ModifySchemasRequest{
			ServiceId: serviceIdDev1,
			Schemas:   schemas,
		})

		log.Info("add schemas")
		schemas = []*pb.Schema{
			{
				SchemaId: "second_schemaId_service_ms",
				Schema:   "second_schema_service_ms",
				Summary:  "second0summary_service_ms",
			},
		}
		respCreateSchema, err = datasource.Instance().ModifySchemas(getContext(), &pb.ModifySchemasRequest{
			ServiceId: serviceIdDev1,
			Schemas:   schemas,
		})
		assert.NoError(t, err)
		assert.Equal(t, proto.Response_SUCCESS, respCreateSchema.Response.GetCode())

		log.Info("query service by serviceID to obtain schema info")
		respGetService, err := datasource.Instance().GetService(getContext(), &pb.GetServiceRequest{
			ServiceId: serviceIdDev1,
		})
		assert.NoError(t, err)
		assert.Equal(t, proto.Response_SUCCESS, respGetService.Response.GetCode())
		assert.Equal(t, []string{"second_schemaId_service_ms"}, respGetService.Service.Schemas)

		log.Info("add new schemaId not exist in service's schemaId list")
		schemas = []*pb.Schema{
			{
				SchemaId: "second_schemaId_service_ms",
				Schema:   "second_schema_service_ms",
				Summary:  "second0summary_service_ms",
			},
		}
		respCreateSchema, err = datasource.Instance().ModifySchemas(getContext(), &pb.ModifySchemasRequest{
			ServiceId: serviceIdDev2,
			Schemas:   schemas,
		})
		assert.NoError(t, err)
		assert.Equal(t, proto.Response_SUCCESS, respCreateSchema.Response.GetCode())

		respGetService, err = datasource.Instance().GetService(getContext(), &pb.GetServiceRequest{
			ServiceId: serviceIdDev2,
		})
		assert.NoError(t, err)
		assert.Equal(t, proto.Response_SUCCESS, respGetService.Response.GetCode())
		assert.Equal(t, []string{"second_schemaId_service_ms"}, respGetService.Service.Schemas)
	})

	t.Run("batch create schemas in production env", func(t *testing.T) {
		var (
			serviceIdPro1 string
			serviceIdPro2 string
		)

		log.Info("register service")
		respCreateService, err := datasource.Instance().RegisterService(getContext(), &pb.CreateServiceRequest{
			Service: &pb.MicroService{
				AppId:       "create_schemas_prod_service_ms",
				ServiceName: "create_schemas_service_service_ms",
				Version:     "1.0.0",
				Level:       "FRONT",
				Status:      pb.MS_UP,
				Environment: pb.ENV_PROD,
			},
		})
		assert.NoError(t, err)
		assert.Equal(t, proto.Response_SUCCESS, respCreateService.Response.GetCode())
		serviceIdPro1 = respCreateService.ServiceId

		respCreateService, err = datasource.Instance().RegisterService(getContext(), &pb.CreateServiceRequest{
			Service: &pb.MicroService{
				AppId:       "create_schemas_prod_service_ms",
				ServiceName: "create_schemas_service_service_ms",
				Version:     "1.0.1",
				Level:       "FRONT",
				Schemas: []string{
					"first_schemaId_service_ms",
					"second_schemaId_service_ms",
				},
				Status:      pb.MS_UP,
				Environment: pb.ENV_PROD,
			},
		})
		assert.NoError(t, err)
		assert.Equal(t, proto.Response_SUCCESS, respCreateService.Response.GetCode())
		serviceIdPro2 = respCreateService.ServiceId

		log.Info("add schemas to service whose schemaId set is empty")
		schemas := []*pb.Schema{
			{
				SchemaId: "first_schemaId_service_ms",
				Schema:   "first_schema_service_ms",
				Summary:  "first0summary_service_ms",
			},
			{
				SchemaId: "first_schemaId_service_ms",
				Schema:   "first_schema_service_ms",
				Summary:  "first0summary_service_ms",
			},
		}
		respModifySchemas, err := datasource.Instance().ModifySchemas(getContext(), &pb.ModifySchemasRequest{
			ServiceId: serviceIdPro1,
			Schemas:   schemas,
		})
		assert.NoError(t, err)
		assert.Equal(t, proto.Response_SUCCESS, respModifySchemas.Response.GetCode())
		respGetService, err := datasource.Instance().GetService(getContext(), &pb.GetServiceRequest{
			ServiceId: serviceIdPro1,
		})
		assert.NoError(t, err)
		assert.Equal(t, proto.Response_SUCCESS, respGetService.Response.GetCode())
		assert.Equal(t, []string{"first_schemaId_service_ms"}, respGetService.Service.Schemas)

		// todo: finish ut after implementing GetAllSchemaInfo, refer to schema_test.go line. 496

		log.Info("modify schemas content already exists, will skip more exist schema")
		respModifySchemas, err = datasource.Instance().ModifySchemas(getContext(), &pb.ModifySchemasRequest{
			ServiceId: serviceIdPro1,
			Schemas:   schemas,
		})
		assert.NoError(t, err)
		assert.Equal(t, proto.Response_SUCCESS, respModifySchemas.Response.GetCode())

		log.Info("add schemas, non-exist schemaId")
		schemas = []*pb.Schema{
			{
				SchemaId: "second_schemaId_service_ms",
				Schema:   "second_schema_service_ms",
				Summary:  "second0summary_service_ms",
			},
		}
		respModifySchemas, err = datasource.Instance().ModifySchemas(getContext(), &pb.ModifySchemasRequest{
			ServiceId: serviceIdPro1,
			Schemas:   schemas,
		})
		assert.NoError(t, err)
		assert.Equal(t, scerr.ErrUndefinedSchemaID, respModifySchemas.Response.GetCode())

		log.Info("add schema when summary is empty")
		respModifySchema, err := datasource.Instance().ModifySchema(getContext(), &pb.ModifySchemaRequest{
			ServiceId: serviceIdPro2,
			SchemaId:  "first_schemaId_service_ms",
			Schema:    "first_schema_service_ms",
		})
		assert.NoError(t, err)
		assert.Equal(t, proto.Response_SUCCESS, respModifySchema.Response.GetCode())

		log.Info("add schemas when summary in database is empty")
		schemas = []*pb.Schema{
			{
				SchemaId: "first_schemaId_service_ms",
				Schema:   "first_schema_service_ms",
				Summary:  "first0summary_service_ms",
			},
		}
		respModifySchemas, err = datasource.Instance().ModifySchemas(getContext(), &pb.ModifySchemasRequest{
			ServiceId: serviceIdPro2,
			Schemas:   schemas,
		})
		assert.NoError(t, err)
		assert.Equal(t, proto.Response_SUCCESS, respModifySchemas.Response.GetCode())
		respExist, err := datasource.Instance().ExistSchema(getContext(), &pb.GetExistenceRequest{
			Type:      service.ExistTypeSchema,
			ServiceId: serviceIdPro2,
			SchemaId:  "first_schemaId_service_ms",
		})
		assert.NoError(t, err)
		assert.Equal(t, "first0summary_service_ms", respExist.Summary)

		respModifySchemas, err = datasource.Instance().ModifySchemas(getContext(), &pb.ModifySchemasRequest{
			ServiceId: serviceIdPro2,
			Schemas:   schemas,
		})
		assert.NoError(t, err)
		assert.Equal(t, proto.Response_SUCCESS, respModifySchemas.Response.GetCode())
	})

	t.Run("create a schema in dev env", func(t *testing.T) {
		var (
			serviceIdDev1 string
			serviceIdDev2 string
		)

		log.Info("register service")
		respCreateService, err := datasource.Instance().RegisterService(getContext(), &pb.CreateServiceRequest{
			Service: &pb.MicroService{
				AppId:       "create_schema_dev_service_ms",
				ServiceName: "create_schema_service_service_ms",
				Version:     "1.0.0",
				Level:       "FRONT",
				Status:      pb.MS_UP,
				Environment: pb.ENV_DEV,
			},
		})
		assert.NoError(t, err)
		assert.Equal(t, proto.Response_SUCCESS, respCreateService.Response.GetCode())
		serviceIdDev1 = respCreateService.ServiceId

		respCreateService, err = datasource.Instance().RegisterService(getContext(), &pb.CreateServiceRequest{
			Service: &pb.MicroService{
				AppId:       "create_schema_dev_service_ms",
				ServiceName: "create_schema_service_service_ms",
				Version:     "1.0.1",
				Level:       "FRONT",
				Schemas: []string{
					"first_schemaId_service_ms",
				},
				Status:      pb.MS_UP,
				Environment: pb.ENV_DEV,
			},
		})
		assert.NoError(t, err)
		assert.Equal(t, proto.Response_SUCCESS, respCreateService.Response.GetCode())
		serviceIdDev2 = respCreateService.ServiceId

		log.Info("create a schema for service whose schemaID is empty")
		respModifySchema, err := datasource.Instance().ModifySchema(getContext(), &pb.ModifySchemaRequest{
			ServiceId: serviceIdDev1,
			SchemaId:  "first_schemaId_service_ms",
			Schema:    "first_schema_service_ms",
		})
		assert.NoError(t, err)
		assert.Equal(t, proto.Response_SUCCESS, respModifySchema.Response.GetCode())

		log.Info("create schema for the service whose schemaId already exist")
		respModifySchema, err = datasource.Instance().ModifySchema(getContext(), &pb.ModifySchemaRequest{
			ServiceId: serviceIdDev2,
			SchemaId:  "first_schemaId_service_ms",
			Schema:    "first_schema_service_ms",
		})
		assert.NoError(t, err)
		assert.Equal(t, proto.Response_SUCCESS, respModifySchema.Response.GetCode())

		log.Info("create schema for the service whose schema summary is empty")
		respModifySchema, err = datasource.Instance().ModifySchema(getContext(), &pb.ModifySchemaRequest{
			ServiceId: serviceIdDev1,
			SchemaId:  "first_schemaId_service_ms",
			Schema:    "first_schema_change_service_ms",
			Summary:   "first0summary1change_service_ms",
		})
		assert.NoError(t, err)
		assert.Equal(t, proto.Response_SUCCESS, respModifySchema.Response.GetCode())

		log.Info("create schema for the service whose schema summary already exist")
		respModifySchema, err = datasource.Instance().ModifySchema(getContext(), &pb.ModifySchemaRequest{
			ServiceId: serviceIdDev1,
			SchemaId:  "first_schemaId_service_ms",
			Schema:    "first_schema_service_ms",
			Summary:   "first0summary_service_ms",
		})
		assert.NoError(t, err)
		assert.Equal(t, proto.Response_SUCCESS, respModifySchema.Response.GetCode())

		log.Info("add schema")
		respModifySchema, err = datasource.Instance().ModifySchema(getContext(), &pb.ModifySchemaRequest{
			ServiceId: serviceIdDev1,
			SchemaId:  "second_schemaId_service_ms",
			Schema:    "second_schema_service_ms",
		})
		assert.NoError(t, err)
		assert.Equal(t, proto.Response_SUCCESS, respModifySchema.Response.GetCode())

	})

	t.Run("create a schema in production env", func(t *testing.T) {
		var (
			serviceIdPro1 string
			serviceIdPro2 string
		)

		log.Info("register service")
		respCreateService, err := datasource.Instance().RegisterService(getContext(), &pb.CreateServiceRequest{
			Service: &pb.MicroService{
				AppId:       "create_schema_prod_service_ms",
				ServiceName: "create_schema_service_service_ms",
				Version:     "1.0.0",
				Level:       "FRONT",
				Status:      pb.MS_UP,
				Environment: pb.ENV_PROD,
			},
		})
		assert.NoError(t, err)
		assert.Equal(t, proto.Response_SUCCESS, respCreateService.Response.GetCode())
		serviceIdPro1 = respCreateService.ServiceId

		respCreateService, err = datasource.Instance().RegisterService(getContext(), &pb.CreateServiceRequest{
			Service: &pb.MicroService{
				AppId:       "create_schema_prod_service_ms",
				ServiceName: "create_schema_service_service_ms",
				Version:     "1.0.1",
				Level:       "FRONT",
				Schemas: []string{
					"first_schemaId_service_ms",
					"second_schemaId_service_ms",
				},
				Status:      pb.MS_UP,
				Environment: pb.ENV_PROD,
			},
		})
		assert.NoError(t, err)
		assert.Equal(t, proto.Response_SUCCESS, respCreateService.Response.GetCode())
		serviceIdPro2 = respCreateService.ServiceId

		log.Info("create a schema for service whose schemaID is empty")
		respModifySchema, err := datasource.Instance().ModifySchema(getContext(), &pb.ModifySchemaRequest{
			ServiceId: serviceIdPro1,
			SchemaId:  "first_schemaId_service_ms",
			Schema:    "first_schema_service_ms",
		})
		assert.NoError(t, err)
		assert.Equal(t, proto.Response_SUCCESS, respModifySchema.Response.GetCode())

		log.Info("modify schema for the service whose schema summary is empty")
		respModifySchema, err = datasource.Instance().ModifySchema(getContext(), &pb.ModifySchemaRequest{
			ServiceId: serviceIdPro1,
			SchemaId:  "first_schemaId_service_ms",
			Schema:    "first_schema_change_service_ms",
			Summary:   "first0summary1change_service_ms",
		})
		assert.NoError(t, err)
		assert.Equal(t, proto.Response_SUCCESS, respModifySchema.Response.GetCode())

		log.Info("modify schema for the service whose schema summary already exist")
		respModifySchema, err = datasource.Instance().ModifySchema(getContext(), &pb.ModifySchemaRequest{
			ServiceId: serviceIdPro1,
			SchemaId:  "first_schemaId_service_ms",
			Schema:    "first_schema_service_ms",
			Summary:   "first0summary_service_ms",
		})
		assert.NoError(t, err)
		assert.NotEqual(t, proto.Response_SUCCESS, respModifySchema.Response.GetCode())

		log.Info("add schema")
		respModifySchema, err = datasource.Instance().ModifySchema(getContext(), &pb.ModifySchemaRequest{
			ServiceId: serviceIdPro1,
			SchemaId:  "second_schemaId_service_ms",
			Schema:    "second_schema_service_ms",
		})
		assert.NoError(t, err)
		assert.NotEqual(t, proto.Response_SUCCESS, respModifySchema.Response.GetCode())

		log.Info("modify schema for the service whose schemaId already exist")
		respModifySchema, err = datasource.Instance().ModifySchema(getContext(), &pb.ModifySchemaRequest{
			ServiceId: serviceIdPro2,
			SchemaId:  "first_schemaId_service_ms",
			Schema:    "first_schema_service_ms",
		})
		assert.NoError(t, err)
		assert.Equal(t, proto.Response_SUCCESS, respModifySchema.Response.GetCode())

	})

	t.Run("create a schema in empty env", func(t *testing.T) {
		var (
			serviceIdPro1 string
			serviceIdPro2 string
		)

		log.Info("register service")
		respCreateService, err := datasource.Instance().RegisterService(getContext(), &pb.CreateServiceRequest{
			Service: &pb.MicroService{
				AppId:       "create_schema_empty_service_ms",
				ServiceName: "create_schema_service_service_ms",
				Version:     "1.0.0",
				Level:       "FRONT",
				Status:      pb.MS_UP,
			},
		})
		assert.NoError(t, err)
		assert.Equal(t, proto.Response_SUCCESS, respCreateService.Response.GetCode())
		serviceIdPro1 = respCreateService.ServiceId

		respCreateService, err = datasource.Instance().RegisterService(getContext(), &pb.CreateServiceRequest{
			Service: &pb.MicroService{
				AppId:       "create_schema_empty_service_ms",
				ServiceName: "create_schema_service_service_ms",
				Version:     "1.0.1",
				Level:       "FRONT",
				Schemas: []string{
					"first_schemaId_service_ms",
					"second_schemaId_service_ms",
				},
				Status: pb.MS_UP,
			},
		})
		assert.NoError(t, err)
		assert.Equal(t, proto.Response_SUCCESS, respCreateService.Response.GetCode())
		serviceIdPro2 = respCreateService.ServiceId

		log.Info("create a schema for service whose schemaID is empty")
		respModifySchema, err := datasource.Instance().ModifySchema(getContext(), &pb.ModifySchemaRequest{
			ServiceId: serviceIdPro1,
			SchemaId:  "first_schemaId_service_ms",
			Schema:    "first_schema_service_ms",
		})
		assert.NoError(t, err)
		assert.Equal(t, proto.Response_SUCCESS, respModifySchema.Response.GetCode())

		log.Info("modify schema for the service whose schema summary is empty")
		respModifySchema, err = datasource.Instance().ModifySchema(getContext(), &pb.ModifySchemaRequest{
			ServiceId: serviceIdPro1,
			SchemaId:  "first_schemaId_service_ms",
			Schema:    "first_schema_change_service_ms",
			Summary:   "first0summary1change_service_ms",
		})
		assert.NoError(t, err)
		assert.Equal(t, proto.Response_SUCCESS, respModifySchema.Response.GetCode())

		log.Info("modify schema for the service whose schema summary already exist")
		respModifySchema, err = datasource.Instance().ModifySchema(getContext(), &pb.ModifySchemaRequest{
			ServiceId: serviceIdPro1,
			SchemaId:  "first_schemaId_service_ms",
			Schema:    "first_schema_service_ms",
			Summary:   "first0summary_service_ms",
		})
		assert.NoError(t, err)
		assert.NotEqual(t, proto.Response_SUCCESS, respModifySchema.Response.GetCode())

		log.Info("add schema")
		respModifySchema, err = datasource.Instance().ModifySchema(getContext(), &pb.ModifySchemaRequest{
			ServiceId: serviceIdPro1,
			SchemaId:  "second_schemaId_service_ms",
			Schema:    "second_schema_service_ms",
		})
		assert.NoError(t, err)
		assert.NotEqual(t, proto.Response_SUCCESS, respModifySchema.Response.GetCode())

		log.Info("modify schema for the service whose schemaId already exist")
		respModifySchema, err = datasource.Instance().ModifySchema(getContext(), &pb.ModifySchemaRequest{
			ServiceId: serviceIdPro2,
			SchemaId:  "first_schemaId_service_ms",
			Schema:    "first_schema_service_ms",
		})
		assert.NoError(t, err)
		assert.Equal(t, proto.Response_SUCCESS, respModifySchema.Response.GetCode())
	})

	t.Run("add a schemaId in production env while schema editable is set", func(t *testing.T) {
		var (
			serviceIdPro1 string
		)
		log.Info("register service")
		respCreateService, err := datasource.Instance().RegisterService(getContext(), &pb.CreateServiceRequest{
			Service: &pb.MicroService{
				AppId:       "add_a_schemaId_prod_schema_lock_ms",
				ServiceName: "add_a_schemaId_prod_schema_lock_service_ms",
				Version:     "1.0.0",
				Level:       "FRONT",
				Status:      pb.MS_UP,
				Environment: pb.ENV_PROD,
			},
		})
		assert.NoError(t, err)
		assert.Equal(t, proto.Response_SUCCESS, respCreateService.Response.GetCode())
		serviceIdPro1 = respCreateService.ServiceId

		log.Info("add a schema with new schemaId, should pass")
		schemas := []*pb.Schema{
			{
				SchemaId: "first_schemaId_ms",
				Schema:   "first_schema_ms",
				Summary:  "first0summary_ms",
			},
		}
		respModifySchemas, err := datasource.Instance().ModifySchemas(getContext(), &pb.ModifySchemasRequest{
			ServiceId: serviceIdPro1,
			Schemas:   schemas,
		})
		assert.NoError(t, err)
		assert.Equal(t, proto.Response_SUCCESS, respModifySchemas.Response.GetCode())

		respService, err := datasource.Instance().GetService(getContext(), &pb.GetServiceRequest{
			ServiceId: serviceIdPro1,
		})
		assert.NoError(t, err)
		assert.Equal(t, proto.Response_SUCCESS, respService.Response.GetCode())
		assert.Equal(t, []string{"first_schemaId_ms"}, respService.Service.Schemas)

		schemas = []*pb.Schema{
			{
				SchemaId: "second_schemaId_ms",
				Schema:   "second_schema_ms",
				Summary:  "second0summary_ms",
			},
		}
		log.Info("schema edit not allowed, add a schema with new schemaId should fail")

		localMicroServiceDs := &DataSource{SchemaEditable: false}
		respModifySchemas, err = localMicroServiceDs.ModifySchemas(getContext(), &pb.ModifySchemasRequest{
			ServiceId: serviceIdPro1,
			Schemas:   schemas,
		})
		assert.NoError(t, err)
		assert.Equal(t, scerr.ErrUndefinedSchemaID, respModifySchemas.Response.GetCode())

		log.Info("schema edit allowed, add a schema with new schemaId, should pass")
		localMicroServiceDs = &DataSource{SchemaEditable: true}
		respModifySchemas, err = localMicroServiceDs.ModifySchemas(getContext(), &pb.ModifySchemasRequest{
			ServiceId: serviceIdPro1,
			Schemas:   schemas,
		})
		assert.NoError(t, err)
		assert.Equal(t, proto.Response_SUCCESS, respModifySchemas.Response.GetCode())
	})

	t.Run("modify a schema in production env while schema editable is set", func(t *testing.T) {
		var (
			serviceIdPro1 string
		)
		log.Info("register service")
		respCreateService, err := datasource.Instance().RegisterService(getContext(), &pb.CreateServiceRequest{
			Service: &pb.MicroService{
				AppId:       "modify_a_schema_prod_schema_lock_ms",
				ServiceName: "modify_a_schema_prod_schema_lock_service_ms",
				Version:     "1.0.0",
				Level:       "FRONT",
				Status:      pb.MS_UP,
				Environment: pb.ENV_PROD,
			},
		})
		assert.NoError(t, err)
		assert.Equal(t, proto.Response_SUCCESS, respCreateService.Response.GetCode())
		serviceIdPro1 = respCreateService.ServiceId

		log.Info("add schemas, should pass")
		schemas := []*pb.Schema{
			{
				SchemaId: "first_schemaId_ms",
				Schema:   "first_schema_ms",
				Summary:  "first0summary_ms",
			},
		}
		respModifySchemas, err := datasource.Instance().ModifySchemas(getContext(), &pb.ModifySchemasRequest{
			ServiceId: serviceIdPro1,
			Schemas:   schemas,
		})
		assert.NoError(t, err)
		assert.Equal(t, proto.Response_SUCCESS, respModifySchemas.Response.GetCode())

		respService, err := datasource.Instance().GetService(getContext(), &pb.GetServiceRequest{
			ServiceId: serviceIdPro1,
		})
		assert.NoError(t, err)
		assert.Equal(t, []string{"first_schemaId_ms"}, respService.Service.Schemas)

		log.Info("schema edit not allowed, modify schema should fail")
		localMicroServiceDs := &DataSource{SchemaEditable: false}
		respModifySchema, err := localMicroServiceDs.ModifySchema(getContext(), &pb.ModifySchemaRequest{
			ServiceId: serviceIdPro1,
			SchemaId:  schemas[0].SchemaId,
			Summary:   schemas[0].Summary,
			Schema:    schemas[0].SchemaId,
		})
		assert.NoError(t, err)
		assert.Equal(t, scerr.ErrModifySchemaNotAllow, respModifySchema.Response.GetCode())

		log.Info("schema edit allowed, add a schema with new schemaId, should pass")
		localMicroServiceDs = &DataSource{SchemaEditable: true}
		respModifySchema, err = localMicroServiceDs.ModifySchema(getContext(), &pb.ModifySchemaRequest{
			ServiceId: serviceIdPro1,
			SchemaId:  schemas[0].SchemaId,
			Summary:   schemas[0].Summary,
			Schema:    schemas[0].SchemaId,
		})
		assert.NoError(t, err)
		assert.Equal(t, proto.Response_SUCCESS, respModifySchema.Response.GetCode())
	})
}

func TestSchema_Exist(t *testing.T) {
	datasource.Install("etcd", func(opts datasource.Options) (datasource.DataSource, error) {
		return NewDataSource(opts), nil
	})
	err := datasource.Init(datasource.Options{
		Endpoint:       "",
		PluginImplName: datasource.ImplName(archaius.GetString("servicecomb.datasource.name", "etcd")),
	})
	assert.NoError(t, err)

	var (
		serviceId string
	)

	t.Run("register service and add schema", func(t *testing.T) {
		log.Info("register service")
		respCreateService, err := datasource.Instance().RegisterService(getContext(), &pb.CreateServiceRequest{
			Service: &pb.MicroService{
				AppId:       "query_schema_group_ms",
				ServiceName: "query_schema_service_ms",
				Version:     "1.0.0",
				Level:       "FRONT",
				Status:      pb.MS_UP,
				Environment: pb.ENV_DEV,
			},
		})
		assert.NoError(t, err)
		assert.Equal(t, proto.Response_SUCCESS, respCreateService.Response.GetCode())
		serviceId = respCreateService.ServiceId

		log.Info("add schemas, should pass")
		resp, err := datasource.Instance().ModifySchema(getContext(), &pb.ModifySchemaRequest{
			ServiceId: serviceId,
			SchemaId:  "com.huawei.test.ms",
			Schema:    "query schema ms",
			Summary:   "summary_ms",
		})
		assert.NoError(t, err)
		assert.Equal(t, proto.Response_SUCCESS, resp.Response.GetCode())

		resp, err = datasource.Instance().ModifySchema(getContext(), &pb.ModifySchemaRequest{
			ServiceId: serviceId,
			SchemaId:  "com.huawei.test.no.summary.ms",
			Schema:    "query schema ms",
		})
		assert.NoError(t, err)
		assert.Equal(t, proto.Response_SUCCESS, resp.Response.GetCode())
	})

	t.Run("check exists", func(t *testing.T) {
		log.Info("check schema exist, should pass")
		resp, err := datasource.Instance().ExistSchema(getContext(), &pb.GetExistenceRequest{
			Type:      service.ExistTypeSchema,
			ServiceId: serviceId,
			SchemaId:  "com.huawei.test.ms",
		})
		assert.NoError(t, err)
		assert.Equal(t, proto.Response_SUCCESS, resp.Response.GetCode())
		assert.Equal(t, "summary_ms", resp.Summary)

		resp, err = datasource.Instance().ExistSchema(getContext(), &pb.GetExistenceRequest{
			Type:        service.ExistTypeSchema,
			ServiceId:   serviceId,
			SchemaId:    "com.huawei.test.ms",
			AppId:       "()",
			ServiceName: "",
			Version:     "()",
		})
		assert.NoError(t, err)
		assert.Equal(t, proto.Response_SUCCESS, resp.Response.GetCode())

		resp, err = datasource.Instance().ExistSchema(getContext(), &pb.GetExistenceRequest{
			Type:      service.ExistTypeSchema,
			ServiceId: serviceId,
			SchemaId:  "com.huawei.test.no.summary.ms",
		})
		assert.NoError(t, err)
		assert.Equal(t, proto.Response_SUCCESS, resp.Response.GetCode())
		assert.Equal(t, "com.huawei.test.no.summary.ms", resp.SchemaId)
		assert.Equal(t, "", resp.Summary)
	})
}
