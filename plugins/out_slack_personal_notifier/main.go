package main

import (
	"C"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"
	"unsafe"

	"github.com/fluent/fluent-bit-go/output"
	"github.com/slack-go/slack"
)

const (
	pluginName = "slack_personal_notifier"
	pluginDesc = "Send personalized direct messages via Slack"
	printDebug = false // для отладки, выставить в `false` при сборке на проде
)

type logWriter struct {
}

func (writer logWriter) Write(bytes []byte) (int, error) {
	level := "" // в уровне логирования выдерживается паддинг в 5 символов, как и логере Fluent Bit
	msg := string(bytes)

	switch {
	case strings.HasPrefix(msg, "panic: "):
		level = "error"
		bytes = []byte(strings.TrimPrefix(msg, "panic: "))
	case strings.HasPrefix(msg, "warning: "):
		level = " warn"
		bytes = []byte(strings.TrimPrefix(msg, "warning: "))
	case strings.HasPrefix(msg, "debug: "):
		level = "debug"
		bytes = []byte(strings.TrimPrefix(msg, "debug: "))
	default:
		level = " info"
		bytes = []byte(strings.TrimPrefix(msg, "info: "))
	}

	if level == "debug" && !printDebug {
		return 0, nil
	}

	return fmt.Printf("[%v] [%s] [%s] %s",
		time.Now().Format("2006/01/02 15:04:05"),
		level,
		pluginName,
		string(bytes),
	)
}

func init() {
	log.SetFlags(0)
	log.SetOutput(new(logWriter))
}

type slackCfg struct {
	token   string
	users   map[string]string
	userKey string
}

//export FLBPluginRegister
func FLBPluginRegister(ctx unsafe.Pointer) int {
	log.Print("registering plugin")
	return output.FLBPluginRegister(ctx, pluginName, pluginDesc)
}

//export FLBPluginExit
func FLBPluginExit() int {
	log.Print("exit plugin")
	return output.FLB_OK
}

//export FLBPluginInit
func FLBPluginInit(plugin unsafe.Pointer) int {
	token := output.FLBPluginConfigKey(plugin, "token")
	users := output.FLBPluginConfigKey(plugin, "users")
	userKey := output.FLBPluginConfigKey(plugin, "user_key")

	if token == "" || users == "" || userKey == "" {
		log.Print("panic: fields 'token', 'users' and 'user_key' cannot be empty")
		return output.FLB_ERROR
	}

	var usersMap map[string]string
	if err := json.Unmarshal([]byte(users), &usersMap); err != nil {
		log.Printf("panic: %s", err)
		return output.FLB_ERROR
	}

	log.Printf("token=***, users=%q", usersMap)

	output.FLBPluginSetContext(plugin, slackCfg{
		token:   token,
		users:   usersMap,
		userKey: userKey,
	})

	return output.FLB_OK
}

//export FLBPluginFlushCtx
func FLBPluginFlushCtx(ctx, data unsafe.Pointer, length C.int, tag *C.char) int {
	cfg := output.FLBPluginGetContext(ctx).(slackCfg)
	dec := output.NewDecoder(data, int(length))

	for {
		ret, ts, record := output.GetRecord(dec)
		if ret != 0 {
			break
		}

		timestamp := convertRawTimestamp(ts)
		message := prepareMessage(timestamp, record)

		recipient, err := extractRecipient(cfg.userKey, record)
		if err != nil {
			log.Printf("debug: %v", err)
			continue // пропускаем сообщение, если оно не содержит поле отправителя
		}

		id, err := getIdByRecipient(recipient, cfg.users)
		if err != nil {
			log.Printf("warning: %v", err)
			continue // если идентификатор не задан
		}

		if err := sendSlackMessage(cfg.token, id, message); err != nil {
			log.Printf("warning: %v", err)
			return output.FLB_RETRY
		}

		log.Printf("sent direct message to recipient '%s' with id '%s'", recipient, id)
	}

	return output.FLB_OK
}

func sendSlackMessage(token, id, text string) error {
	api := slack.New(token)

	_, _, err := api.PostMessage(
		id,
		slack.MsgOptionText(text, false),
		slack.MsgOptionAsUser(true),
	)
	if err != nil {
		return fmt.Errorf("failed to send message to recipient with id '%s': %w", id, err)
	}

	return nil
}

func prepareMessage(ts time.Time, record map[any]any) string {
	strBuilder := strings.Builder{}

	strBuilder.WriteString(fmt.Sprintf("[\"timestamp\": %d.000000000, {", ts.Unix()))

	countOfRecords := len(record)
	processedRecords := 0

	for k, v := range record {
		strBuilder.WriteString(fmt.Sprintf("\"%s\"=>\"%s\"", k, v))

		processedRecords++

		if processedRecords < countOfRecords {
			strBuilder.WriteString(", ")
		}
	}

	strBuilder.WriteString("}]")

	return strBuilder.String()
}

func convertRawTimestamp(ts any) time.Time {
	var timestamp time.Time

	switch t := ts.(type) {
	case output.FLBTime:
		timestamp = t.Time
	case uint64:
		timestamp = time.Unix(int64(t), 0)
	default:
		log.Print("warning: timestamp isn't known format, use current time")

		timestamp = time.Now()
	}

	return timestamp
}

func extractRecipient(key string, record map[any]any) (string, error) {
	if recipient, ok := record[key]; ok {
		if strRecipient, ok := recipient.([]byte); ok {
			return string(strRecipient), nil
		}

		return "", fmt.Errorf("failed to convert recipient '%v' to text format", recipient)
	}

	return "", fmt.Errorf("unable to identify recipient by using key '%s' from the message body", key)
}

func getIdByRecipient(recipient string, users map[string]string) (string, error) {
	if id, ok := users[recipient]; ok {
		return id, nil
	}

	return "", fmt.Errorf("failed to retrieve id for recipient '%s'", recipient)
}

func main() {
}
