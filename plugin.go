package main

import (
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/bluele/slack"
	"github.com/drone/drone-template-lib/template"
)

type (
	Repo struct {
		Owner string
		Name  string
	}

	Build struct {
		Tag      string
		Event    string
		Number   int
		Commit   string
		Ref      string
		Branch   string
		Author   Author
		Pull     string
		Message  Message
		DeployTo string
		Status   string
		Link     string
		Started  int64
		Created  int64
	}

	Author struct {
		Username string
		Name     string
		Email    string
		Avatar   string
	}

	Message struct {
		msg   string
		Title string
		Body  string
	}

	Config struct {
		Webhook        string
		Channel        string
		Recipient      string
		Username       string
		Template       string
		AttachmentFile string
		ImageURL       string
		IconURL        string
		IconEmoji      string
		LinkNames      bool
	}

	Job struct {
		Started int64
	}

	Plugin struct {
		Repo   Repo
		Build  Build
		Config Config
		Job    Job
	}
)

func (a Author) String() string {
	return a.Username
}

func newCommitMessage(m string) Message {
	// not checking the length here
	// as split will always return at least one element
	// check it if using more than the first element
	splitMsg := strings.Split(m, "\n")

	return Message{
		msg:   m,
		Title: strings.TrimSpace(splitMsg[0]),
		Body:  strings.TrimSpace(strings.Join(splitMsg[1:], "\n")),
	}
}
func (m Message) String() string {
	return m.msg
}

func (p Plugin) Exec() error {
	attachment := slack.Attachment{
		Fallback:   fallback(p.Repo, p.Build),
		Color:      color(p.Build),
		MarkdownIn: []string{"text", "fallback"},
		ImageURL:   p.Config.ImageURL,
	}

	payload := slack.WebHookPostPayload{}
	payload.Username = p.Config.Username
	payload.Attachments = []*slack.Attachment{&attachment}
	payload.IconUrl = p.Config.IconURL
	payload.IconEmoji = p.Config.IconEmoji

	if p.Config.Recipient != "" {
		payload.Channel = prepend("@", p.Config.Recipient)
	} else if p.Config.Channel != "" {
		payload.Channel = prepend("#", p.Config.Channel)
	}
	if p.Config.LinkNames == true {
		payload.LinkNames = "1"
	}
	if p.Config.Template != "" {
		var err error
		attachment.Text, err = templateMessage(p.Config.Template, p)
		if err != nil {
			return err
		}
	} else {
		attachment.Text = message(p.Repo, p.Build)
	}

	if p.Config.AttachmentFile != "" {
		dat, err := ioutil.ReadFile(p.Config.AttachmentFile)
		if err != nil {
			return err
		}
		attachment.Text = fmt.Sprintf("%s\n%s", attachment.Text, string(dat))
	}

	client := slack.NewWebHook(p.Config.Webhook)
	return client.PostMessage(&payload)
}

func templateMessage(t string, plugin Plugin) (string, error) {
	return template.RenderTrim(t, plugin)
}

func message(repo Repo, build Build) string {
	return fmt.Sprintf("*%s* <%s|%s/%s#%s> (%s) by %s",
		build.Status,
		build.Link,
		repo.Owner,
		repo.Name,
		build.Commit[:8],
		build.Branch,
		build.Author,
	)
}

func fallback(repo Repo, build Build) string {
	return fmt.Sprintf("%s %s/%s#%s (%s) by %s",
		build.Status,
		repo.Owner,
		repo.Name,
		build.Commit[:8],
		build.Branch,
		build.Author,
	)
}

func color(build Build) string {
	switch build.Status {
	case "success":
		return "good"
	case "failure", "error", "killed":
		return "danger"
	default:
		return "warning"
	}
}

func prepend(prefix, s string) string {
	if !strings.HasPrefix(s, prefix) {
		return prefix + s
	}

	return s
}
