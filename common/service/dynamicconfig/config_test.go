// Copyright (c) 2017 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package dynamicconfig

import (
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/suite"
	"github.com/uber-common/bark"
)

type inMemoryClient struct {
	globalValues atomic.Value
}

func newInMemoryClient() *inMemoryClient {
	var globalValues atomic.Value
	globalValues.Store(make(map[Key]interface{}))
	return &inMemoryClient{globalValues: globalValues}
}

func (mc *inMemoryClient) SetValue(key Key, value interface{}) {
	v := mc.globalValues.Load().(map[Key]interface{})
	v[key] = value
	mc.globalValues.Store(v)
}

func (mc *inMemoryClient) GetValue(key Key, defaultValue interface{}) (interface{}, error) {
	v := mc.globalValues.Load().(map[Key]interface{})
	if val, ok := v[key]; ok {
		return val, nil
	}
	return defaultValue, errors.New("unable to find key")
}

func (mc *inMemoryClient) GetValueWithFilters(
	name Key, filters map[Filter]interface{}, defaultValue interface{},
) (interface{}, error) {
	return mc.GetValue(name, defaultValue)
}

func (mc *inMemoryClient) GetIntValue(name Key, filters map[Filter]interface{}, defaultValue int) (int, error) {
	v := mc.globalValues.Load().(map[Key]interface{})
	if val, ok := v[name]; ok {
		return val.(int), nil
	}
	return defaultValue, errors.New("unable to find key")
}

func (mc *inMemoryClient) GetFloatValue(name Key, filters map[Filter]interface{}, defaultValue float64) (float64, error) {
	v := mc.globalValues.Load().(map[Key]interface{})
	if val, ok := v[name]; ok {
		return val.(float64), nil
	}
	return defaultValue, errors.New("unable to find key")
}

func (mc *inMemoryClient) GetBoolValue(name Key, filters map[Filter]interface{}, defaultValue bool) (bool, error) {
	v := mc.globalValues.Load().(map[Key]interface{})
	if val, ok := v[name]; ok {
		return val.(bool), nil
	}
	return defaultValue, errors.New("unable to find key")
}

func (mc *inMemoryClient) GetStringValue(name Key, filters map[Filter]interface{}, defaultValue string) (string, error) {
	v := mc.globalValues.Load().(map[Key]interface{})
	if val, ok := v[name]; ok {
		return val.(string), nil
	}
	return defaultValue, errors.New("unable to find key")
}

func (mc *inMemoryClient) GetMapValue(
	name Key, filters map[Filter]interface{}, defaultValue map[string]interface{},
) (map[string]interface{}, error) {
	v := mc.globalValues.Load().(map[Key]interface{})
	if val, ok := v[name]; ok {
		return val.(map[string]interface{}), nil
	}
	return defaultValue, errors.New("unable to find key")
}

func (mc *inMemoryClient) GetDurationValue(
	name Key, filters map[Filter]interface{}, defaultValue time.Duration,
) (time.Duration, error) {
	v := mc.globalValues.Load().(map[Key]interface{})
	if val, ok := v[name]; ok {
		return val.(time.Duration), nil
	}
	return defaultValue, errors.New("unable to find key")
}

type configSuite struct {
	suite.Suite
	client *inMemoryClient
	cln    *Collection
}

func TestConfigSuite(t *testing.T) {
	s := new(configSuite)
	suite.Run(t, s)
}

func (s *configSuite) SetupSuite() {
	s.client = newInMemoryClient()
	s.cln = NewCollection(s.client, bark.NewLoggerFromLogrus(logrus.New()))
}

func (s *configSuite) TestGetProperty() {
	key := testGetPropertyKey
	value := s.cln.GetProperty(key, "a")
	s.Equal("a", value())
	s.client.SetValue(key, "b")
	s.Equal("b", value())
}

func (s *configSuite) TestGetIntProperty() {
	key := testGetIntPropertyKey
	value := s.cln.GetIntProperty(key, 10)
	s.Equal(10, value())
	s.client.SetValue(key, 50)
	s.Equal(50, value())
}

func (s *configSuite) TestGetFloat64Property() {
	key := testGetFloat64PropertyKey
	value := s.cln.GetFloat64Property(key, 0.1)
	s.Equal(0.1, value())
	s.client.SetValue(key, 0.01)
	s.Equal(0.01, value())
}

func (s *configSuite) TestGetBoolProperty() {
	key := testGetBoolPropertyKey
	value := s.cln.GetBoolProperty(key, true)
	s.Equal(true, value())
	s.client.SetValue(key, false)
	s.Equal(false, value())
}

func (s *configSuite) TestGetDurationProperty() {
	key := testGetDurationPropertyKey
	value := s.cln.GetDurationProperty(key, time.Second)
	s.Equal(time.Second, value())
	s.client.SetValue(key, time.Minute)
	s.Equal(time.Minute, value())
}
