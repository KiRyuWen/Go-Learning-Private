package queue

import "strings"

type JobType string

const (
	JobTypeBahamut JobType = "bahamut"
	JobTypeYouTube JobType = "youtube"
)

const (
	DiscordTaskQueue = "tasks:discord:messages"
)

type DiscordMessage struct {
	ID string `json:"id"` //for different job index

	Type JobType `json:"type"`

	ChannelID string `json:"channel_id"`

	TargetURL string `json:"target_url"`
}

func NewDiscordMessage(msgID string, jobType JobType, channelID, targetURL string) *DiscordMessage {
	return &DiscordMessage{
		ID:        msgID,
		Type:      jobType,
		ChannelID: channelID,
		TargetURL: targetURL,
	}
}

func GetMsgType(msg string) (JobType, bool) {
	if strings.Contains(msg, "https://forum.gamer.com.tw/") {
		return JobTypeBahamut, true
	}

	return "", false
}
