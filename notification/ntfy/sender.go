package ntfy

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/target/goalert/config"
	"github.com/target/goalert/notification"
)

type Sender struct{}

type NTFYEnumActionType string
const (
	NTFYEnumActionTypeView NTFYEnumActionType = "view"
	NTFYEnumActionTypeBroadcast NTFYEnumActionType = "broadcast"
	NTFYEnumActionTypeHttp NTFYEnumActionType = "http"
)

type NTFYActionButton struct {
	Type NTFYEnumActionType
	Text string
	Parameters []string
	Clear bool
}

type NTFYBaseData struct {
	Title string
	Priority string
	Tags []string
	Message string
	Click string
}

func NewSender(ctx context.Context) *Sender {
	return &Sender{}
}

// Send will send an alert for the provided message type
func (s *Sender) Send(ctx context.Context, msg notification.Message) (*notification.SentMessage, error) {
	cfg := config.FromContext(ctx)
	baseUrl := cfg.NTFY.BaseURL
	var payload NTFYBaseData
	switch m := msg.(type) {
	case notification.Test:
		payload = NTFYBaseData{
			Title: "Test Notification",
			Priority: "default",
			Tags: []string{"toolbox"},
			Message: "This is a test notification from GoAlert.",
			Click: cfg.CallbackURL("/"),
		}
	case notification.Verification:
		payload = NTFYBaseData{
			Title: "Verification",
			Priority: "default",
			Tags: []string{"computer"},
			Message: "Your verification code is " + strconv.Itoa(m.Code) + ".",
			Click: cfg.CallbackURL("/verify"),
		}
	case notification.Alert:
		payload = NTFYBaseData{
			Title: m.Summary,
			Priority: "urgent",
			Tags: []string{"rotating_light"},
			Message: m.Details,
			Click: cfg.CallbackURL("/alerts/" + strconv.Itoa(m.AlertID)),
		}
	case notification.AlertBundle:
		payload = NTFYBaseData{
			Title: m.ServiceName,
			Priority: "urgent",
			Tags: []string{"rotating_light"},
			Message: m.ServiceName + "\n" + strconv.Itoa(m.Count) + " alerts",
			Click: cfg.CallbackURL("/services/" + m.ServiceID + "/alerts"),
		}
	case notification.AlertStatus:
		payload = NTFYBaseData{
			Title: "Alert Status: " + strconv.Itoa(m.AlertID),
			Priority: "default",
			Tags: []string{"page_with_curl"},
			Message: "Alert Status: " + strconv.Itoa(m.AlertID) + "\n" + m.LogEntry,
			Click: cfg.CallbackURL("/alerts/" + strconv.Itoa(m.AlertID)),
		}
	case notification.ScheduleOnCallUsers:
		users := make([]string, len(m.Users))
		for i, u := range m.Users {
			users[i] = u.Name
		}
		payload = NTFYBaseData{
			Title: "On Call: " + m.ScheduleName,
			Priority: "default",
			Tags: []string{"calendar"},
			Message: "On Call: " + m.ScheduleName + "\n" + "Users on call: " + strings.Join(users, ",") + "\n" + m.ScheduleURL,
			Click: cfg.CallbackURL("/schedules/" + m.ScheduleID),
		}
	default:
		return nil, fmt.Errorf("message type '%s' not supported", m.Type().String())
	}

	ctx, cancel := context.WithTimeout(ctx, time.Second*3)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", baseUrl + "/" + msg.Destination().Value, strings.NewReader(payload.Message))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Title", payload.Title)
	req.Header.Set("Priority", payload.Priority)
	req.Header.Set("Tags", strings.Join(payload.Tags, ","))
	req.Header.Set("Click", payload.Click)

	_, err = http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	return &notification.SentMessage{State: notification.StateSent}, nil
}
