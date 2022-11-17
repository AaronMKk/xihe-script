package main

import (
	"time"

	"github.com/opensourceways/xihe-grpc-protocol/grpc/client"
	"github.com/opensourceways/xihe-grpc-protocol/grpc/competition"
	"github.com/qinsheng99/go-py-message/app"
	"github.com/qinsheng99/go-py-message/config"
	"github.com/qinsheng99/go-py-message/infrastructure/message"
	"github.com/sirupsen/logrus"
)

type handler struct {
	log       *logrus.Entry
	maxRetry  int
	evaluate  app.EvaluateService
	calculate app.CalculateService
	match     config.MatchImpl
	cli       *client.CompetitionClient
}

type handlerMessage struct {
	message.MatchMessage
	score  float32
	status string
}

const sleepTime = 100 * time.Millisecond

func (h *handler) Calculate(cal *message.MatchMessage, match *message.MatchFields) error {
	return h.do(func(b bool) error {
		var res message.ScoreRes
		var m = handlerMessage{MatchMessage: *cal}
		err := h.calculate.Calculate(match, &res)
		if err != nil {
			h.log.Errorf("calculate script failed,err: %v", err)
			m.status = err.Error()
		} else {
			if res.Status != 200 {
				m.status = res.Msg
			} else {
				m.score = res.Data
			}
		}
		h.handlerCompetition(m)
		return err
	})
}

func (h *handler) Evaluate(eval *message.MatchMessage, match *message.MatchFields) error {
	return h.do(func(b bool) error {
		var res message.ScoreRes
		var m = handlerMessage{MatchMessage: *eval}
		err := h.evaluate.Evaluate(match, &res)
		if err != nil {
			h.log.Errorf("evaluate script failed,err: %v", err)
			m.status = err.Error()
		} else {
			if res.Status != 200 {
				m.status = res.Msg
			} else {
				m.score = res.Metrics.Acc
			}
		}
		h.handlerCompetition(m)
		return err
	})
}

func (h *handler) GetMatch(id string) message.MatchFieldImpl {
	return h.match.GetMatch(id)
}

func (h *handler) handlerCompetition(m handlerMessage) {
	err := h.cli.SetSubmissionInfo(&competition.CompetitionIndex{
		Id:    m.CompetitionId,
		Phase: m.Phase,
	}, &competition.SubmissionInfo{
		Id:     m.UserId,
		Status: m.status,
		Score:  m.score,
	})
	if err != nil {
		h.log.Errorf("call competition rpc failed,err :%v ,data:%v", err, m)
	} else {
		h.log.Debugf("call competition rpc, id:%s,user:%v,stage:%v,res:(%s/%v)", m.CompetitionId, m.UserId, m.Phase, m.status, m.score)
	}
}

func (h *handler) do(f func(bool) error) (err error) {
	return h.retry(f, sleepTime)
}

func (h *handler) retry(f func(bool) error, interval time.Duration) (err error) {
	n := h.maxRetry - 1

	if err = f(n <= 0); err == nil || n <= 0 {
		return
	}

	for i := 1; i < n; i++ {
		time.Sleep(interval)

		if err = f(false); err == nil {
			return
		}
	}

	time.Sleep(interval)

	return f(true)
}
