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

package persistence

import (
	"os"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	//"github.com/pborman/uuid"
	"github.com/pborman/uuid"
	gen "github.com/uber/cadence/.gen/go/shared"
	"github.com/uber/cadence/common"
)

type (
	historyPersistenceSuite struct {
		suite.Suite
		TestBase
		// override suite.Suite.Assertions with require.Assertions; this means that s.NotNil(nil) will stop the test,
		// not merely log an error
		*require.Assertions
	}
)

func TestHistoryPersistenceSuite(t *testing.T) {
	s := new(historyPersistenceSuite)
	suite.Run(t, s)
}

func (s *historyPersistenceSuite) SetupSuite() {
	if testing.Verbose() {
		log.SetOutput(os.Stdout)
	}

	s.SetupWorkflowStore()
}

func (s *historyPersistenceSuite) SetupTest() {
	// Have to define our overridden assertions in the test setup. If we did it earlier, s.T() will return nil
	s.Assertions = require.New(s.T())
}

func (s *historyPersistenceSuite) TearDownSuite() {
	s.TearDownWorkflowStore()
}

func (s *historyPersistenceSuite) TestAppendHistoryEvents() {
	domainID := "ff03c29f-fcf1-4aea-893d-1a7ec421e3ec"
	workflowExecution := gen.WorkflowExecution{
		WorkflowId: common.StringPtr("append-history-events-test"),
		RunId:      common.StringPtr("986fc9cd-4a2d-4964-bf9f-5130116d5851"),
	}

	events1 := []byte("event1;event2")
	serializedHistory := &SerializedHistoryEventBatch{Version: 1, EncodingType: common.EncodingTypeJSON, Data: events1}
	err0 := s.AppendHistoryEvents(domainID, workflowExecution, 1, common.EmptyVersion, 1, 1, serializedHistory, false)
	s.Nil(err0)

	events2 := []byte("event3;")
	serializedHistory.Data = events2
	err1 := s.AppendHistoryEvents(domainID, workflowExecution, 3, common.EmptyVersion, 1, 1, serializedHistory, false)
	s.Nil(err1)

	events2New := []byte("event3new;")
	serializedHistory.Data = events2New
	err2 := s.AppendHistoryEvents(domainID, workflowExecution, 3, common.EmptyVersion, 1, 1, serializedHistory, false)
	s.NotNil(err2)
	s.IsType(&ConditionFailedError{}, err2)

	err3 := s.AppendHistoryEvents(domainID, workflowExecution, 3, common.EmptyVersion, 1, 2, serializedHistory, true)
	s.Nil(err3)
}

func (s *historyPersistenceSuite) TestGetHistoryEvents() {
	domainID := "0fdc53ef-b890-4870-a944-b9b028ac9742"
	workflowExecution := gen.WorkflowExecution{
		WorkflowId: common.StringPtr("get-history-events-test"),
		RunId:      common.StringPtr("26fa29f6-af41-4b70-9a3b-8b1b35eed82a"),
	}

	batchEvents := newBatchEventForTest([]int64{1, 2}, 1)
	err0 := s.AppendHistoryEvents(domainID, workflowExecution, 1, common.EmptyVersion, 1, 1, batchEvents.batch, false)
	s.Nil(err0)

	history, token, err1 := s.GetWorkflowExecutionHistory(domainID, workflowExecution, 1, 2, 10, nil)
	s.Nil(err1)
	s.Equal(0, len(token))
	s.Equal(2, len(history.Events))
	s.Equal(int64(1), history.Events[0].GetVersion())
}

type testBatchEvent struct {
	batch  *SerializedHistoryEventBatch
	events []*gen.HistoryEvent
}

func newBatchEventForTest(eventIDs []int64, version int64) *testBatchEvent {
	var events []*gen.HistoryEvent
	for _, eid := range eventIDs {
		e := &gen.HistoryEvent{EventId: common.Int64Ptr(eid), Version: common.Int64Ptr(version)}
		events = append(events, e)
	}

	historySerializer := NewJSONHistorySerializer()
	batch, err := historySerializer.Serialize(NewHistoryEventBatch(GetDefaultHistoryVersion(), events))
	if err != nil {
		panic(err)
	}
	return &testBatchEvent{batch: batch, events: events}
}

func (s *historyPersistenceSuite) TestGetHistoryEventsCompatibility() {
	domainID := "373de9d6-e41e-42d4-bee9-9e06968e4d0d"
	workflowExecution := gen.WorkflowExecution{
		WorkflowId: common.StringPtr("get-history-events-compatibility-test"),
		RunId:      common.StringPtr(uuid.New()),
	}

	batches := []*testBatchEvent{
		newBatchEventForTest([]int64{1, 2}, 1),
		newBatchEventForTest([]int64{3}, 1),
		newBatchEventForTest([]int64{4, 5}, 1),
		newBatchEventForTest([]int64{5}, 1), // staled batch, should be ignored
		newBatchEventForTest([]int64{6, 7}, 1),
	}

	for i, be := range batches {
		err0 := s.AppendHistoryEvents(domainID, workflowExecution, be.events[0].GetEventId(), common.EmptyVersion, 1, int64(i), be.batch, false)
		s.Nil(err0)
	}

	// pageSize=3, get 3 batches
	history, token, err := s.GetWorkflowExecutionHistory(domainID, workflowExecution, 1, 8, 3, nil)
	s.Nil(err)
	s.NotNil(token)
	s.Equal(5, len(history.Events))
	for i, e := range history.Events {
		s.Equal(int64(i+1), e.GetEventId())
	}

	// get next page, should ignore staled batch
	history, token, err = s.GetWorkflowExecutionHistory(domainID, workflowExecution, 1, 8, 3, token)
	s.Nil(err)
	s.Nil(token)
	s.Equal(2, len(history.Events))
	s.Equal(int64(6), history.Events[0].GetEventId())
	s.Equal(int64(7), history.Events[1].GetEventId())
}

func (s *historyPersistenceSuite) TestDeleteHistoryEvents() {
	domainID := "373de9d6-e41e-42d4-bee9-9e06968e4d0d"
	workflowExecution := gen.WorkflowExecution{
		WorkflowId: common.StringPtr("delete-history-events-test"),
		RunId:      common.StringPtr("2122fd8d-f583-459e-a2e2-d1fb273a43cb"),
	}

	events := []*testBatchEvent{
		newBatchEventForTest([]int64{1, 2}, 1),
		newBatchEventForTest([]int64{3}, 1),
		newBatchEventForTest([]int64{4, 5}, 1),
		newBatchEventForTest([]int64{5}, 1), // staled batch, should be ignored
		newBatchEventForTest([]int64{6, 7}, 1),
	}
	for i, be := range events {
		err0 := s.AppendHistoryEvents(domainID, workflowExecution, be.events[0].GetEventId(), common.EmptyVersion, 1, int64(i), be.batch, false)
		s.Nil(err0)
	}

	history, token, err1 := s.GetWorkflowExecutionHistory(domainID, workflowExecution, 1, 10, 11, nil)
	s.Nil(err1)
	s.Nil(token)
	s.Equal(7, len(history.Events))
	for i, e := range history.Events {
		s.Equal(int64(i+1), e.GetEventId())
	}

	err2 := s.DeleteWorkflowExecutionHistory(domainID, workflowExecution)
	s.Nil(err2)

	data1, token1, err3 := s.GetWorkflowExecutionHistory(domainID, workflowExecution, 1, 10, 11, nil)
	s.NotNil(err3)
	s.IsType(&gen.EntityNotExistsError{}, err3)
	s.Nil(token1)
	s.Nil(data1)
}

func (s *historyPersistenceSuite) TestAppendAndGet() {
	domainID := uuid.New()
	workflowExecution := gen.WorkflowExecution{
		WorkflowId: common.StringPtr("append-and-get-test"),
		RunId:      common.StringPtr(uuid.New()),
	}
	batches := []*testBatchEvent{
		newBatchEventForTest([]int64{1, 2}, 0),
		newBatchEventForTest([]int64{3, 4}, 1),
		newBatchEventForTest([]int64{5, 6}, 2),
		newBatchEventForTest([]int64{7, 8}, 3),
	}

	for i := 0; i < len(batches); i++ {

		err0 := s.AppendHistoryEvents(domainID, workflowExecution, batches[i].events[0].GetEventId(), common.EmptyVersion, 1, int64(i), batches[i].batch, false)
		s.Nil(err0)

		history, token, err1 := s.GetWorkflowExecutionHistory(domainID, workflowExecution, 1, 10, 11, nil)
		s.Nil(err1)
		s.Nil(token)
		s.Equal((i+1)*2, len(history.Events))

		for j, e := range history.Events {
			s.Equal(int64(j+1), e.GetEventId())
		}
	}
}

func (s *historyPersistenceSuite) TestOverwriteAndShadowingHistoryEvents() {
	domainID := "003de9c6-e41e-42d4-bee9-9e06968e4d0d"
	workflowExecution := gen.WorkflowExecution{
		WorkflowId: common.StringPtr("delete-history-partial-events-test"),
		RunId:      common.StringPtr("2122fd8d-2859-459e-a2e2-d1fb273a43cb"),
	}
	version1 := int64(123)
	version2 := int64(1234)
	var err error

	eventBatches := []*testBatchEvent{
		newBatchEventForTest([]int64{1, 2}, 1),
		newBatchEventForTest([]int64{3}, 1),
		newBatchEventForTest([]int64{4, 5}, 1),
		newBatchEventForTest([]int64{6}, 1),
		newBatchEventForTest([]int64{7}, 1),
		newBatchEventForTest([]int64{8, 9}, 1),
		newBatchEventForTest([]int64{10}, 1),
		newBatchEventForTest([]int64{11, 12}, 1),
		newBatchEventForTest([]int64{13}, 1),
		newBatchEventForTest([]int64{14}, 1),
	}

	for i, be := range eventBatches {
		err = s.AppendHistoryEvents(domainID, workflowExecution, be.events[0].GetEventId(), version1, 1, int64(i), be.batch, false)
		s.Nil(err)
	}

	history, token, err := s.GetWorkflowExecutionHistory(domainID, workflowExecution, 1, 25, 25, nil)
	s.Nil(err)
	s.Nil(token)
	s.Equal(14, len(history.Events))
	for i, e := range history.Events {
		s.Equal(int64(i+1), e.GetEventId())
	}

	newEventBatchs := []*testBatchEvent{
		newBatchEventForTest([]int64{8, 9, 10, 11, 12}, 1),
		newBatchEventForTest([]int64{13, 14, 15, 16}, 1),
		newBatchEventForTest([]int64{17, 18}, 1),
		newBatchEventForTest([]int64{19, 20, 21, 22, 23}, 1),
		newBatchEventForTest([]int64{24}, 1),
	}

	for _, be := range newEventBatchs {
		override := false
		for _, oe := range eventBatches {
			if oe.events[0].GetEventId() == be.events[0].GetEventId() {
				override = true
				break
			}
		}
		err = s.AppendHistoryEvents(domainID, workflowExecution, be.events[0].GetEventId(), version2, 1, 999, be.batch, override)
		s.Nil(err)
	}
	historyEvents := []*gen.HistoryEvent{}
	token = nil
	for {
		history, token, err = s.GetWorkflowExecutionHistory(domainID, workflowExecution, 1, 25, 3, token)
		s.Nil(err)
		historyEvents = append(historyEvents, history.Events...)
		if len(token) == 0 {
			break
		}
	}
	s.Empty(token)
	s.Equal(24, len(historyEvents))
	for i, e := range historyEvents {
		s.Equal(int64(i+1), e.GetEventId())
	}
}

func (s *historyPersistenceSuite) AppendHistoryEvents(domainID string, workflowExecution gen.WorkflowExecution,
	firstEventID, eventBatchVersion int64, rangeID, txID int64, eventsBatch *SerializedHistoryEventBatch, overwrite bool) error {

	return s.HistoryMgr.AppendHistoryEvents(&AppendHistoryEventsRequest{
		DomainID:          domainID,
		Execution:         workflowExecution,
		FirstEventID:      firstEventID,
		EventBatchVersion: eventBatchVersion,
		RangeID:           rangeID,
		TransactionID:     txID,
		Events:            eventsBatch,
		Overwrite:         overwrite,
	})
}

func (s *historyPersistenceSuite) GetWorkflowExecutionHistory(domainID string, workflowExecution gen.WorkflowExecution,
	firstEventID, nextEventID int64, pageSize int, token []byte) (*gen.History, []byte, error) {

	response, err := s.HistoryMgr.GetWorkflowExecutionHistory(&GetWorkflowExecutionHistoryRequest{
		DomainID:      domainID,
		Execution:     workflowExecution,
		FirstEventID:  firstEventID,
		NextEventID:   nextEventID,
		PageSize:      pageSize,
		NextPageToken: token,
	})

	if err != nil {
		return nil, nil, err
	}

	return response.History, response.NextPageToken, nil
}

func (s *historyPersistenceSuite) DeleteWorkflowExecutionHistory(domainID string,
	workflowExecution gen.WorkflowExecution) error {

	return s.HistoryMgr.DeleteWorkflowExecutionHistory(&DeleteWorkflowExecutionHistoryRequest{
		DomainID:  domainID,
		Execution: workflowExecution,
	})
}
