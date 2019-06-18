package mongodb

import (
	"bufio"
	"fmt"
	"github.com/annchain/BlockDB/common/bytes"
	//"github.com/annchain/BlockDB/processors"
	"github.com/sirupsen/logrus"
	"io"
	"net"
	"time"
)

const headerLen = 16

type MongoProcessor struct {
	config MongoProcessorConfig

	readPool  *Pool
	writePool *Pool
}
type MongoProcessorConfig struct {
	IdleConnectionTimeout time.Duration
}

func (m *MongoProcessor) Stop() {
	logrus.Info("MongoProcessor stopped")
}

func (m *MongoProcessor) Start() {
	logrus.Info("MongoProcessor started")
	// start consuming queue
}

func NewMongoProcessor(config MongoProcessorConfig) *MongoProcessor {
	//TODO move mongo url into config
	url := "172.28.152.101:27017"

	return &MongoProcessor{
		config:    config,
		readPool:  NewPool(url,10),
		writePool: NewPool(url,10),
	}
}

func (m *MongoProcessor) ProcessConnection(conn net.Conn) error {
	defer conn.Close()

	fmt.Println("start process connection")
	reader := bufio.NewReader(conn)

	// 1, parse command
	// 2, dispatch the command to every interested parties
	//    including chain logger and the real backend mongoDB server
	// 3, response to conn
	for {
		conn.SetReadDeadline(time.Now().Add(m.config.IdleConnectionTimeout))

		cmdHeader := make([]byte, headerLen)
		_, err := reader.Read(cmdHeader)
		if err != nil {
			if err == io.EOF {
				fmt.Println("target closed")
				logrus.Info("target closed")
				return nil
			} else if neterr, ok := err.(net.Error); ok && neterr.Timeout() {
				fmt.Println("target timeout")
				logrus.Info("target timeout")
				conn.Close()
				return nil
			}
			return err
		}

		// query command
		msgSize := bytes.GetInt32(cmdHeader, 0)
		fmt.Println("msgsize: ", msgSize)

		cmdBody := make([]byte, msgSize-headerLen)
		_, err = reader.Read(cmdBody)
		if err != nil {
			fmt.Println("read body error: ", err)
			return err
		}
		fmt.Println(fmt.Sprintf("msg header: %x", cmdHeader))
		fmt.Println(fmt.Sprintf("msg body: %x", cmdBody))

		cmdFull := append(cmdHeader, cmdBody...)
		err = m.messageHandler(cmdFull, conn)
		if err != nil {
			// TODO handle err
			return err
		}

	}
	return nil
}

func (m *MongoProcessor) messageHandler(bytes []byte, client net.Conn) error {

	var msg RequestMessage
	err := msg.Decode(bytes)
	if err != nil {
		// TODO handle err
		return err
	}

	var pool *Pool
	if msg.ReadOnly() {
		pool = m.readPool
	} else {
		pool = m.writePool
	}
	server := pool.Acquire()
	defer pool.Release(server)

	err = msg.WriteTo(server)
	if err != nil {
		// TODO handle err
		return err
	}

	var msgResp ResponseMessage
	err = msgResp.ReadFromMongo(server)
	if err != nil {
		// TODO handle err
		return err
	}
	err = msgResp.WriteTo(client)
	if err != nil {
		// TODO handle err
		return err
	}

	err = m.handleBlockDBEvents(&msgResp)
	if err != nil {
		// TODO handle err
		return err
	}

	return nil
}

func (m *MongoProcessor) handleBlockDBEvents(msg MongoMessage) error {
	// TODO not implemented yet

	events := msg.ParseCommand()

	fmt.Println("block db events: ", events)

	return nil
}