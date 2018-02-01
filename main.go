package main

import (
	"crypto/rand"
	"fmt"
	"math"
	"strconv"
	"strings"

	smpp "github.com/CodeMonkeyKevin/smpp34"
	gsmutil "github.com/CodeMonkeyKevin/smpp34/gsmutil"
	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	host     = kingpin.Flag("host", "SMPP host name with port. Example: \"localhost:9000\"").Short('h').Default("localhost:9000").String()
	username = kingpin.Flag("username", "SMPP username.").Short('u').Default("smpp").String()
	password = kingpin.Flag("password", "SMPP password.").Short('p').Default("smpp").String()
	message  = kingpin.Flag("message", "SMS Message").Short('m').Required().String()
	from     = kingpin.Flag("from", "SMS Message From.").Short('f').Default("sms-send").String()
	to       = kingpin.Flag("to", "SMS Message To.").Short('t').Required().String()
)

func main() {
	kingpin.Version("Version: " + version + "\nBuildTime: " + buildTime + "\nCommit: " + commit)
	kingpin.Parse()

	s := strings.Split(*host, ":")
	hostname := s[0]
	port := 9000
	if len(s) > 1 {
		port, _ = strconv.Atoi(s[1])
	}
	fmt.Printf("Connecting to %v:%v\n", hostname, port)
	// connect and bind
	tx, err := smpp.NewTransceiver(
		hostname,
		port,
		5,
		smpp.Params{
			"system_type": "CMT",
			"system_id":   *username,
			"password":    *password,
		},
	)
	if err != nil {
		fmt.Println("Connection Err:", err)
		return
	}

	smBytes := gsmutil.EncodeUcs2(*message)
	smLen := len(smBytes)
	fmt.Println("Message Bytes count:", smLen)

	if smLen > 140 {
		totalParts := byte(int(math.Ceil(float64(smLen) / 134.0)))
		sendParams := smpp.Params{smpp.DATA_CODING: smpp.ENCODING_ISO10646, smpp.ESM_CLASS: smpp.ESM_CLASS_GSMFEAT_UDHI}
		partNum := 1
		uid := make([]byte, 1)
		_, err := rand.Read(uid)
		if err != nil {
			// fmt.Println("QuerySM error:", err)
			fmt.Println("Rand.Read error:", err)
			return
		}
		for i := 0; i < smLen; i += 134 {
			start := i
			end := i + 134
			if end > smLen {
				end = smLen
			}
			part := []byte{0x05, 0x00, 0x03, uid[0], totalParts, byte(partNum)}
			part = append(part, smBytes[start:end]...)
			fmt.Println("Part:", part)
			// Send SubmitSm
			seq, err := tx.SubmitSmEncoded(*from, *to, part, &sendParams)
			// Pdu gen errors
			if err != nil {
				fmt.Println("SubmitSm err:", err)
			}
			// Should save this to match with message_id
			fmt.Println("seq:", seq)
			partNum++

		}

	} else {
		sendParams := smpp.Params{}
		// Send SubmitSm
		seq, err := tx.SubmitSm(*from, *to, *message, &sendParams)

		// Pdu gen errors
		if err != nil {
			fmt.Println("SubmitSm err:", err)
		}
		// Should save this to match with message_id
		fmt.Println("seq:", seq)

	}

	for {
		pdu, err := tx.Read() // This is blocking
		if err != nil {
			fmt.Println("Read Err:", err)
			break
		}

		// EnquireLinks are auto handles
		switch pdu.GetHeader().Id {
		case smpp.SUBMIT_SM_RESP:
			// message_id should match this with seq message
			fmt.Println("MSG ID:", pdu.GetField("message_id").Value())
			fmt.Printf("PDU Header: %v", pdu.GetHeader())
			fmt.Println()
		default:
			// ignore all other PDUs or do what you link with them
			fmt.Println("PDU ID:", pdu.GetHeader().Id)
		}
	}

	fmt.Println("ending...")
}
