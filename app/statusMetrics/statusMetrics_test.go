/*
 * Copyright (c) 2015 GRNET S.A.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the
 * License. You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an "AS
 * IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
 * express or implied. See the License for the specific language
 * governing permissions and limitations under the License.
 *
 * The views and conclusions contained in the software and
 * documentation are those of the authors and should not be
 * interpreted as representing official policies, either expressed
 * or implied, of GRNET S.A.
 *
 */

package statusMetrics

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ARGOeu/argo-web-api/Godeps/_workspace/src/github.com/gorilla/mux"
	"github.com/ARGOeu/argo-web-api/Godeps/_workspace/src/github.com/stretchr/testify/suite"
	"github.com/ARGOeu/argo-web-api/Godeps/_workspace/src/gopkg.in/gcfg.v1"
	"github.com/ARGOeu/argo-web-api/Godeps/_workspace/src/gopkg.in/mgo.v2"
	"github.com/ARGOeu/argo-web-api/Godeps/_workspace/src/gopkg.in/mgo.v2/bson"
	"github.com/ARGOeu/argo-web-api/respond"
	"github.com/ARGOeu/argo-web-api/utils/authentication"
	"github.com/ARGOeu/argo-web-api/utils/config"
	"github.com/ARGOeu/argo-web-api/utils/mongo"
)

// This is a util. suite struct used in tests (see pkg "testify")
type StatusMetricsTestSuite struct {
	suite.Suite
	cfg          config.Config
	router       *mux.Router
	confHandler  respond.ConfHandler
	tenantDbConf config.MongoConfig
}

// Setup the Test Environment
// This function runs before any test and setups the environment
// A test configuration object is instantiated using a reference
// to testdb: argo_test_details. Also here is are instantiated some expected
// xml response validation messages (authorization,crud responses).
// Also the testdb is seeded with tenants,reports,metric_profiles and status_metrics
func (suite *StatusMetricsTestSuite) SetupTest() {

	const testConfig = `
    [server]
    bindip = ""
    port = 8080
    maxprocs = 4
    cache = false
    lrucache = 700000000
    gzip = true
    [mongodb]
    host = "127.0.0.1"
    port = 27017
    db = "argotest_metrics"
`

	_ = gcfg.ReadStringInto(&suite.cfg, testConfig)

	// Create router and confhandler for test
	suite.confHandler = respond.ConfHandler{suite.cfg}
	suite.router = mux.NewRouter().StrictSlash(true).PathPrefix("/api/v2/status").Subrouter()
	HandleSubrouter(suite.router, &suite.confHandler)

	// Connect to mongo testdb
	session, _ := mongo.OpenSession(suite.cfg.MongoDB)

	// Add authentication token to mongo testdb
	seedAuth := bson.M{"api_key": "S3CR3T"}
	_ = mongo.Insert(session, suite.cfg.MongoDB.Db, "authentication", seedAuth)

	// seed mongo
	session, err := mgo.Dial(suite.cfg.MongoDB.Host)
	if err != nil {
		panic(err)
	}
	defer session.Close()

	// seed a tenant to use
	c := session.DB(suite.cfg.MongoDB.Db).C("tenants")
	c.Insert(bson.M{
		"id": "6ac7d684-1f8e-4a02-a502-720e8f11e50c",
		"info": bson.M{
			"name":    "GUARDIANS",
			"email":   "email@something2",
			"website": "www.gotg.com",
			"created": "2015-10-20 02:08:04",
			"updated": "2015-10-20 02:08:04"},
		"db_conf": []bson.M{
			bson.M{
				"store":    "main",
				"server":   "localhost",
				"port":     27017,
				"database": "argotest_metrics_egi",
				"username": "",
				"password": ""},
		},
		"users": []bson.M{
			bson.M{
				"name":    "egi_user",
				"email":   "egi_user@email.com",
				"api_key": "KEY1"},
		}})

	c.Insert(bson.M{
		"id": "6ac7d684-1f8e-4a02-a502-720e8f11e50d",
		"info": bson.M{
			"name":    "AVENGERS",
			"email":   "email@something2",
			"website": "www.gotg.com",
			"created": "2015-10-20 02:08:04",
			"updated": "2015-10-20 02:08:04"},
		"db_conf": []bson.M{
			bson.M{
				"store":    "main",
				"server":   "localhost",
				"port":     27017,
				"database": "argotest_metrics_tenant2",
				"username": "",
				"password": ""},
		},
		"users": []bson.M{
			bson.M{
				"name":    "tenant2_user",
				"email":   "tenant2_user@email.com",
				"api_key": "KEY2"},
		}})

	// get dbconfiguration based on the tenant
	// Prepare the request object
	request, _ := http.NewRequest("GET", "", strings.NewReader(""))
	// add the content-type header to application/json
	request.Header.Set("Content-Type", "application/json")
	// add the authentication token which is seeded in testdb
	request.Header.Set("x-api-key", "KEY1")
	// authenticate user's api key and find corresponding tenant
	suite.tenantDbConf, err = authentication.AuthenticateTenant(request.Header, suite.cfg)

	// Now seed the report DEFINITIONS
	c = session.DB(suite.tenantDbConf.Db).C("reports")
	c.Insert(bson.M{
		"id": "eba61a9e-22e9-4521-9e47-ecaa4a494364",
		"info": bson.M{
			"name":        "Report_A",
			"description": "report aaaaa",
			"created":     "2015-9-10 13:43:00",
			"updated":     "2015-10-11 13:43:00",
		},
		"topology_schema": bson.M{
			"group": bson.M{
				"type": "NGI",
				"group": bson.M{
					"type": "SITES",
				},
			},
		},
		"profiles": []bson.M{
			bson.M{
				"id":   "6ac7d684-1f8e-4a02-a502-720e8f11e50b",
				"type": "metric",
				"name": "profile1"},
			bson.M{
				"id":   "6ac7d684-1f8e-4a02-a502-720e8f11e523",
				"type": "operations",
				"name": "profile2"},
			bson.M{
				"id":   "6ac7d684-1f8e-4a02-a502-720e8f11e50q",
				"type": "aggregation",
				"name": "profile3"},
		},
		"filter_tags": []bson.M{
			bson.M{
				"name":  "name1",
				"value": "value1"},
			bson.M{
				"name":  "name2",
				"value": "value2"},
		}})

	// seed the status detailed metric data
	c = session.DB(suite.tenantDbConf.Db).C("status_metrics")
	c.Insert(bson.M{
		"report":             "eba61a9e-22e9-4521-9e47-ecaa4a494364",
		"monitoring_box":     "nagios3.hellasgrid.gr",
		"date_integer":       20150501,
		"timestamp":          "2015-05-01T00:00:00Z",
		"service":            "CREAM-CE",
		"host":               "cream01.afroditi.gr",
		"endpoint_group":     "HG-03-AUTH",
		"metric":             "emi.cream.CREAMCE-JobSubmit",
		"status":             "OK",
		"time_integer":       0,
		"previous_state":     "OK",
		"previous_timestamp": "2015-04-30T23:59:00Z",
		"summary":            "Cream status is ok",
		"message":            "Cream job submission test return value of ok",
	})
	c.Insert(bson.M{
		"report":             "eba61a9e-22e9-4521-9e47-ecaa4a494364",
		"monitoring_box":     "nagios3.hellasgrid.gr",
		"date_integer":       20150501,
		"timestamp":          "2015-05-01T01:00:00Z",
		"service":            "CREAM-CE",
		"host":               "cream01.afroditi.gr",
		"endpoint_group":     "HG-03-AUTH",
		"metric":             "emi.cream.CREAMCE-JobSubmit",
		"status":             "CRITICAL",
		"time_integer":       10000,
		"previous_state":     "OK",
		"previous_timestamp": "2015-05-01T00:00:00Z",
		"summary":            "Cream status is CRITICAL",
		"message":            "Cream job submission test failed",
	})
	c.Insert(bson.M{
		"report":             "eba61a9e-22e9-4521-9e47-ecaa4a494364",
		"monitoring_box":     "nagios3.hellasgrid.gr",
		"date_integer":       20150501,
		"timestamp":          "2015-05-01T05:00:00Z",
		"service":            "CREAM-CE",
		"host":               "cream01.afroditi.gr",
		"endpoint_group":     "HG-03-AUTH",
		"metric":             "emi.cream.CREAMCE-JobSubmit",
		"status":             "OK",
		"time_integer":       50000,
		"previous_state":     "CRITICAL",
		"previous_timestamp": "2015-05-01T01:00:00Z",
		"summary":            "Cream status is ok",
		"message":            "Cream job submission test return value of ok",
	})

	// get dbconfiguration based on the tenant
	// Prepare the request object
	request, _ = http.NewRequest("GET", "", strings.NewReader(""))
	// add the content-type header to application/json
	request.Header.Set("Content-Type", "application/json")
	// add the authentication token which is seeded in testdb
	request.Header.Set("x-api-key", "KEY2")
	// authenticate user's api key and find corresponding tenant
	suite.tenantDbConf, err = authentication.AuthenticateTenant(request.Header, suite.cfg)

	// Now seed the reports DEFINITIONS
	c = session.DB(suite.tenantDbConf.Db).C("reports")
	c.Insert(bson.M{
		"id": "eba61a9e-22e9-4521-9e47-ecaa4a494365",
		"info": bson.M{
			"name":        "Report_B",
			"description": "report aaaaa",
			"created":     "2015-9-10 13:43:00",
			"updated":     "2015-10-11 13:43:00",
		},
		"topology_schema": bson.M{
			"group": bson.M{
				"type": "EUDAT_GROUPS",
				"group": bson.M{
					"type": "EUDAT_SITES",
				},
			},
		},
		"profiles": []bson.M{
			bson.M{
				"id":   "6ac7d684-1f8e-4a02-a502-720e8f11e50b",
				"type": "metric",
				"name": "eudat.CRITICAL"},
			bson.M{
				"id":   "6ac7d684-1f8e-4a02-a502-720e8f11e523",
				"type": "operations",
				"name": "profile2"},
			bson.M{
				"id":   "6ac7d684-1f8e-4a02-a502-720e8f11e50q",
				"type": "aggregation",
				"name": "profile3"},
		},
		"filter_tags": []bson.M{
			bson.M{
				"name":  "name1",
				"value": "value1"},
			bson.M{
				"name":  "name2",
				"value": "value2"},
		}})

	// seed the status detailed metric data
	c = session.DB(suite.tenantDbConf.Db).C("status_metrics")
	c.Insert(bson.M{
		"report":             "eba61a9e-22e9-4521-9e47-ecaa4a494365",
		"monitoring_box":     "nagios3.tenant2.eu",
		"date_integer":       20150501,
		"timestamp":          "2015-05-01T00:00:00Z",
		"service":            "someService",
		"host":               "someservice.example.gr",
		"endpoint_group":     "EL-01-AUTH",
		"metric":             "someService-FileTransfer",
		"status":             "OK",
		"time_integer":       0,
		"previous_state":     "OK",
		"previous_timestamp": "2015-04-30T23:59:00Z",
		"summary":            "someService status is ok",
		"message":            "someService data upload test return value of ok",
	})
	c.Insert(bson.M{
		"report":             "eba61a9e-22e9-4521-9e47-ecaa4a494365",
		"monitoring_box":     "nagios3.tenant2.eu",
		"date_integer":       20150501,
		"timestamp":          "2015-05-01T01:00:00Z",
		"service":            "someService",
		"host":               "someservice.example.gr",
		"endpoint_group":     "EL-01-AUTH",
		"metric":             "someService-FileTransfer",
		"status":             "CRITICAL",
		"time_integer":       10000,
		"previous_state":     "OK",
		"previous_timestamp": "2015-05-01T00:00:00Z",
		"summary":            "someService status is CRITICAL",
		"message":            "someService data upload test failed",
	})
	c.Insert(bson.M{
		"report":             "eba61a9e-22e9-4521-9e47-ecaa4a494365",
		"monitoring_box":     "nagios3.tenant2.eu",
		"date_integer":       20150501,
		"timestamp":          "2015-05-01T05:00:00Z",
		"service":            "someService",
		"host":               "someservice.example.gr",
		"endpoint_group":     "EL-01-AUTH",
		"metric":             "someService-FileTransfer",
		"status":             "OK",
		"time_integer":       50000,
		"previous_state":     "CRITICAL",
		"previous_timestamp": "2015-05-01T01:00:00Z",
		"summary":            "someService status is ok",
		"message":            "someService data upload test return value of ok",
	})

}

func (suite *StatusMetricsTestSuite) TestListStatusMetrics() {
	respXML1 := `<root>
 <group name="HG-03-AUTH" type="SITES">
  <group name="CREAM-CE" type="service">
   <endpoint name="cream01.afroditi.gr">
    <metric name="emi.cream.CREAMCE-JobSubmit">
     <status timestamp="2015-04-30T23:59:00Z" value="OK"></status>
     <status timestamp="2015-05-01T00:00:00Z" value="OK"></status>
     <status timestamp="2015-05-01T01:00:00Z" value="CRITICAL"></status>
     <status timestamp="2015-05-01T05:00:00Z" value="OK"></status>
    </metric>
   </endpoint>
  </group>
 </group>
</root>`

	respXML2 := `<root>
 <group name="EL-01-AUTH" type="EUDAT_SITES">
  <group name="someService" type="service">
   <endpoint name="someservice.example.gr">
    <metric name="someService-FileTransfer">
     <status timestamp="2015-04-30T23:59:00Z" value="OK"></status>
     <status timestamp="2015-05-01T00:00:00Z" value="OK"></status>
     <status timestamp="2015-05-01T01:00:00Z" value="CRITICAL"></status>
     <status timestamp="2015-05-01T05:00:00Z" value="OK"></status>
    </metric>
   </endpoint>
  </group>
 </group>
</root>`

	respJSON1 := `{
 "groups": [
  {
   "name": "HG-03-AUTH",
   "type": "SITES",
   "services": [
    {
     "name": "CREAM-CE",
     "type": "service",
     "endpoints": [
      {
       "name": "cream01.afroditi.gr",
       "metrics": [
        {
         "name": "emi.cream.CREAMCE-JobSubmit",
         "statuses": [
          {
           "timestamp": "2015-04-30T23:59:00Z",
           "value": "OK"
          },
          {
           "timestamp": "2015-05-01T00:00:00Z",
           "value": "OK"
          },
          {
           "timestamp": "2015-05-01T01:00:00Z",
           "value": "CRITICAL"
          },
          {
           "timestamp": "2015-05-01T05:00:00Z",
           "value": "OK"
          }
         ]
        }
       ]
      }
     ]
    }
   ]
  }
 ]
}`

	respJSON2 := `{
 "groups": [
  {
   "name": "EL-01-AUTH",
   "type": "EUDAT_SITES",
   "services": [
    {
     "name": "someService",
     "type": "service",
     "endpoints": [
      {
       "name": "someservice.example.gr",
       "metrics": [
        {
         "name": "someService-FileTransfer",
         "statuses": [
          {
           "timestamp": "2015-04-30T23:59:00Z",
           "value": "OK"
          },
          {
           "timestamp": "2015-05-01T00:00:00Z",
           "value": "OK"
          },
          {
           "timestamp": "2015-05-01T01:00:00Z",
           "value": "CRITICAL"
          },
          {
           "timestamp": "2015-05-01T05:00:00Z",
           "value": "OK"
          }
         ]
        }
       ]
      }
     ]
    }
   ]
  }
 ]
}`

	fullurl1 := "/api/v2/status/Report_A/SITES/HG-03-AUTH" +
		"/services/CREAM-CE/endpoints/cream01.afroditi.gr/metrics/emi.cream.CREAMCE-JobSubmit" +
		"?start_time=2015-05-01T00:00:00Z&end_time=2015-05-01T23:00:00Z"

	fullurl2 := "/api/v2/status/Report_B/EUDAT_SITES/EL-01-AUTH" +
		"/services/someService/endpoints/someservice.example.gr/metrics/someService-FileTransfer" +
		"?start_time=2015-05-01T00:00:00Z&end_time=2015-05-01T23:00:00Z"

	fullurl3 := "/api/v2/status/Report_B/EUDAT_SITES/EL-01-AUTH" +
		"/services/someService/endpoints/someservice.example.gr/metrics" +
		"?start_time=2015-05-01T00:00:00Z&end_time=2015-05-01T23:00:00Z"

	// 1. EGI XML REQUEST
	// init the response placeholder
	response := httptest.NewRecorder()
	// Prepare the request object for first tenant
	request, _ := http.NewRequest("GET", fullurl1, strings.NewReader(""))
	// add accept xml header
	request.Header.Set("Accept", "application/xml")
	// add the authentication token which is seeded in testdb
	request.Header.Set("x-api-key", "KEY1")
	// Serve the http request
	suite.router.ServeHTTP(response, request)
	// Check that we must have a 200 ok code
	suite.Equal(200, response.Code, "Internal Server Error")
	// Compare the expected and actual xml response
	suite.Equal(respXML1, response.Body.String(), "Response body mismatch")

	// 2. TENANT2 XML REQUEST
	// init the response placeholder
	response = httptest.NewRecorder()
	// Prepare the request object for second tenant
	request, _ = http.NewRequest("GET", fullurl2, strings.NewReader(""))
	// add accept xml header
	request.Header.Set("Accept", "application/xml")
	// add the authentication token which is seeded in testdb
	request.Header.Set("x-api-key", "KEY2")
	// Serve the http request
	suite.router.ServeHTTP(response, request)
	// Check that we must have a 200 ok code
	suite.Equal(200, response.Code, "Internal Server Error")
	// Compare the expected and actual xml response
	suite.Equal(respXML2, response.Body.String(), "Response body mismatch")

	// 3. EGI JSON REQUEST
	// init the response placeholder
	response = httptest.NewRecorder()
	// Prepare the request object for second tenant
	request, _ = http.NewRequest("GET", fullurl1, strings.NewReader(""))
	// add json accept header
	request.Header.Set("Accept", "application/json")
	// add the authentication token which is seeded in testdb
	request.Header.Set("x-api-key", "KEY1")
	// Serve the http request
	suite.router.ServeHTTP(response, request)
	// Check that we must have a 200 ok code
	suite.Equal(200, response.Code, "Internal Server Error")
	// Compare the expected and actual xml response
	suite.Equal(respJSON1, response.Body.String(), "Response body mismatch")

	// 4. TENANT2 JSON REQUEST
	// init the response placeholder
	response = httptest.NewRecorder()
	// Prepare the request object for second tenant
	request, _ = http.NewRequest("GET", fullurl2, strings.NewReader(""))
	// add json accept header
	request.Header.Set("Accept", "application/json")
	// add the authentication token which is seeded in testdb
	request.Header.Set("x-api-key", "KEY2")
	// Serve the http request
	suite.router.ServeHTTP(response, request)
	// Check that we must have a 200 ok code
	suite.Equal(200, response.Code, "Internal Server Error")
	// Compare the expected and actual xml response
	suite.Equal(respJSON2, response.Body.String(), "Response body mismatch")

	// 5. TENANT2 ALL JSON REQUEST
	// init the response placeholder
	response = httptest.NewRecorder()
	// Prepare the request object for second tenant
	request, _ = http.NewRequest("GET", fullurl3, strings.NewReader(""))
	// add json accept header
	request.Header.Set("Accept", "application/json")
	// add the authentication token which is seeded in testdb
	request.Header.Set("x-api-key", "KEY2")
	// Serve the http request
	suite.router.ServeHTTP(response, request)
	// Check that we must have a 200 ok code
	suite.Equal(200, response.Code, "Internal Server Error")
	// Compare the expected and actual xml response
	suite.Equal(respJSON2, response.Body.String(), "Response body mismatch")

}

// This function is actually called in the end of all tests
// and clears the test environment.
// Mainly it's purpose is to drop the testdb
func (suite *StatusMetricsTestSuite) TearDownTest() {

	session, _ := mongo.OpenSession(suite.cfg.MongoDB)

	session.DB("argotest_metrics").DropDatabase()
	session.DB("argotest_metrics_tenant2").DropDatabase()
	session.DB("argotest_metrics_egi").DropDatabase()
}

// This is the first function called when go test is issued
func TestStatusMetricsSuite(t *testing.T) {
	suite.Run(t, new(StatusMetricsTestSuite))
}
