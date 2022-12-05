package google_chat_alert

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
	"github.com/cloudevents/sdk-go/v2/event"
)

func init() {
	// Register a CloudEvent function with the Functions Framework
	functions.CloudEvent("GoogleChatAlert", googleChatAlert)
}

type messageData struct {
	Message eventData `json:"message"`
}

type eventData struct {
	Data []byte `json:"data"`
}

type messagePayload struct {
	Incident incident `json:"incident"`
}

type documentation struct {
	Content string `json:"content"`
}

type incident struct {
	PolicyName    string        `json:"policy_name"`
	IncidentID    string        `json:"incident_id"`
	Documentation documentation `json:"documentation"`
	State         string        `json:"state"`
	Url           string        `json:"url"`
	StartedAt     int64         `json:"started_at"`
}

type googleChatResponse struct {
	Error googleChatError `json:"error"`
}

type googleChatError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Status  string `json:"status"`
}

const MESSAGE_TEMPLATE = `
{
    "cards": [
        {
            "header": {
                "title": "<users/all> Google Cloud Monitoring Alert"
            },
            "sections": [
                {
                    "header": "{alertName}",
                    "widgets": [
                        {
                            "textParagraph": {
                                "text": "<b>Started at:</b> {startedAt}"
                            }
                        },
                    ]
                },
                {
                    "header": "<b><font color=\"#ff0000\">Received log</font></b>",
                    "widgets": [
                        {
                            "textParagraph": {
                                "text": "{logMessage}"
                            }
                        }
                    ]
                },
                {
                    "widgets": [
                        {
                            "keyValue": {
                                "topLabel": "Status",
                                "content": "{status}"
                            }
                        }
                    ]
                },
                {
                    "widgets": [
                        {
                            "buttons": [
                                {
                                    "textButton": {
                                        "text": "GO TO INCIDENT",
                                        "onClick": {
                                            "openLink": {
                                                "url": "{incidentURL}"
                                            }
                                        }
                                    }
                                }
                            ]
                        }
                    ]
                }
            ]
        }
    ]
}
`

func formatMessage(message string, args ...string) string {
	r := strings.NewReplacer(args...)
	return r.Replace(message)
}

func googleChatAlert(ctx context.Context, e event.Event) error {
	var msg messageData
	err := e.DataAs(&msg)
	if err != nil {
		return fmt.Errorf("failed to retrieve PubSub message: %v", err)
	}

	var pld messagePayload
	err = json.Unmarshal(msg.Message.Data, &pld)
	if err != nil {
		return fmt.Errorf("failed to parse PubSub msg payload: %v", err)
	}

	incident := pld.Incident

	webhookUrl := "<YOUR WEBHOOK URL>"

	argsList := []string{
		"{alertName}", incident.PolicyName,
		"{startedAt}", time.Unix(incident.StartedAt, 0).Format("2006-01-02 15:04:05 CET"),
		"{logMessage}", incident.Documentation.Content,
		"{status}", incident.State,
		"{incidentURL}", incident.Url}
	message := formatMessage(MESSAGE_TEMPLATE, argsList...)

	res, err := http.Post(webhookUrl, "application/json", bytes.NewBuffer([]byte(message)))
	if err != nil {
		return fmt.Errorf("failed to send request to Google Chat: %v", err)
	}

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("failed to read body: %v", err)
	}

	var googleChatRes googleChatResponse
	err = json.Unmarshal(resBody, &googleChatRes)
	if err != nil {
		return fmt.Errorf("failed to unmarshal google chat response: %v, received: %s", err, string(resBody))
	}
	if (googleChatRes.Error != googleChatError{}) {
		return fmt.Errorf("received an error response from Google Chat: %v", googleChatRes)
	}
	return nil
}
