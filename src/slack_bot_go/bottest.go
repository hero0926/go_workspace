// go slack API를 발견...ㅠㅠ
// https://github.com/nlopes/slack
// 위를 이용한 예제
// https://github.com/tcnksm/go-slack-interactive

package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/nlopes/slack"
)

const (
	// action is used for slack attament action.
	actionSelect = "select"
	actionStart  = "start"
	actionCancel = "cancel"
	buttonSelect = "button"
)

type envConfig struct {
	// Port is server port to be listened.
	Port string `envconfig:"PORT" default:"3000"`

	// BotToken is bot user token to access to slack API.
	BotToken string `envconfig:"BOT_TOKEN" required:"true"`

	// VerificationToken is used to validate interactive messages from slack.
	VerificationToken string `envconfig:"VERIFICATION_TOKEN" required:"true"`

	// BotID is bot user ID.
	BotID string `envconfig:"BOT_ID" required:"true"`

	// ChannelID is slack channel ID where bot is working.
	// Bot responses to the mention in this channel.
	ChannelID string `envconfig:"CHANNEL_ID" required:"true"`
}

type twitterConfig struct {
	confKey     string
	confSecret  string
	tokenKey    string
	tokenSecret string
}

type SlackListener struct {
	client    *slack.Client
	botID     string
	channelID string
}

func main() {

	os.Exit(_main(os.Args[1:]))

	
}

func _main(args []string) int {

	getGitCommit("hero0926")

	defer func() {
		if err := recover(); err != nil {
			log.Fatal(err)
			return
		}
	}()

	// 채널형 고루틴으로 변경?????
	// 블로그, 트위터를 그런식으로?????(예비 자료 크롤링?)

	// 유저 메시지 입력 외에도 이벤트 만들만한 것 생각해 보기...

	// 1. 설정
	var env envConfig
	env = envconfig(env)
	api := slack.New(env.BotToken)

	var tweetenv twitterConfig
	tweetenv = twitterconfig(tweetenv)

	slackListener := &SlackListener{
		client:    api,
		botID:     env.BotID,
		channelID: env.ChannelID,
	}

	// DEBUG설정 - 개발시에만 켜주세요
	//api.SetDebug(true)
	//로그인 테스트하기
	groups, err := api.GetGroups(false)
	if err != nil {
		log.Printf("%s 로그인 중 에러가 발생하였습니다. : %s\n", groups, err)
		return 0
	}

	// 2. 메시지 받는 설정

	go slackListener.ListenAndResponse(tweetenv)
	go slackListener.PostByTime(env)

	// 서버를 생성하면 그 주소로 설정하면 됩니다(버튼 클릭 액션을 받아올 때 사용)
	http.Handle("/interaction", interactionHandler{
		verificationToken: env.VerificationToken,
	})

	log.Printf("[INFO] Server listening on :%s", env.Port)
	if err := http.ListenAndServe(":"+env.Port, nil); err != nil {
		log.Printf("[ERROR] %s", err)
		return 1
	}
	return 0

}

// 메시지 받는 기능
func (s *SlackListener) ListenAndResponse(tweetenv twitterConfig) {
	rtm := s.client.NewRTM()
	go rtm.ManageConnection()

	for msg := range rtm.IncomingEvents {

		// 타입이 특정 인터페이스를 구현하는지 검사
		// interface{}.(타입).(구현하는지 궁금한 인터페이스)
		switch ev := msg.Data.(type) {
		/////////////interface.(type) 형식을 눈에 익혀두자~
		//Data 인터페이스의 type에 따라 switch 문 돌리는 중...
		//slack 의 messageEvent일때 처리

		case *slack.MessageEvent:
			if err := s.handleMessageEvent(ev, tweetenv); err != nil {

				log.Printf("[ERROR] 처리중 에러가 발생하였습니다.: %s", err)
			}
		}
	}
}

// 메시지 받고 보내기
func (s *SlackListener) handleMessageEvent(ev *slack.MessageEvent, tweetenv twitterConfig) error {

	userID := ev.Msg.User

	receivedMsg := ev.Msg.Text

	// 다른 채널에 쳤을때
	if ev.Channel != s.channelID {
		log.Printf("다른 채널 : %s %s", ev.Channel, s.channelID)
		return nil
	}

	log.Println(userID, " : ", receivedMsg)

	if strings.Contains(receivedMsg, `¯\_(ツ)_/¯`) {
		if strings.Contains(ev.Msg.Username, "go돌이") {
			log.Println("봇이 한 대화라 무시 했어요.")
			return nil
		}
		s.client.PostMessage(ev.Channel, `¯\_(ツ)_/¯`, slack.PostMessageParameters{})
	}

	// 봇에게 한 멘션이 아닐 때
	if !(strings.HasPrefix(receivedMsg, fmt.Sprintf("<@%s> ", s.botID))) {

		// 봇이 한 말이면 무시하자!
		if strings.Contains(ev.Msg.Username, "go돌이") {
			log.Println("봇이 한 대화라 무시 했어요.")
			return nil
		}

		// 1. 기사 찾기

		if strings.Contains(receivedMsg, "기사") || strings.Contains(receivedMsg, "뉴스") || strings.Contains(receivedMsg, "소식") {

			log.Println("기사 크롤링 시.")
			m := NewsScrape()

			if !(len(m) == 0) {

				for k, v := range m {

					attachment := slack.Attachment{

						Color: "#cc1512",
						Title: k,
						Text:  v,
					}

					params := slack.PostMessageParameters{

						Attachments: []slack.Attachment{
							attachment,
						},
					}

					s.client.PostMessage(ev.Channel, "", params)

				}

			} else {
				s.client.PostMessage(ev.Channel, "알 수 없는 에러가 발생했습니다. 다시 시도해 주세요.", slack.PostMessageParameters{})
			}
		}

		// 2. 오키 게시글 찾기

		if strings.Contains(receivedMsg, "오키") || strings.Contains(receivedMsg, "옼희") {

			log.Println("오키 크롤링 시.")
			m := OkkyScrape()

			if len(m) == 0 {
				s.client.PostMessage(ev.Channel, "알 수 없는 에러가 발생했습니다. 다시 시도해 주세요.", slack.PostMessageParameters{})
			} else {

				for k, v := range m {

					attachment := slack.Attachment{

						Color: "#104293",
						Title: k,
						Text:  v,
					}

					params := slack.PostMessageParameters{

						Attachments: []slack.Attachment{
							attachment,
						},
					}

					s.client.PostMessage(ev.Channel, "", params)

				}
			}

		}

		// 라. 블로그 입력 시(RSS)

		if strings.Contains(receivedMsg, "블로그") {

			log.Println("블로그 크롤링 시.")

			m := RssScrape()

			if len(m) == 0 {
				s.client.PostMessage(ev.Channel, "알 수 없는 에러가 발생했습니다. 다시 시도해 주세요.", slack.PostMessageParameters{})
			} else {

				for k, v := range m {

					attachment := slack.Attachment{

						Color: "#2a4f2e",
						Title: k,
						Text:  v,
					}

					params := slack.PostMessageParameters{

						Attachments: []slack.Attachment{
							attachment,
						},
					}

					s.client.PostMessage(ev.Channel, "", params)

				}
			}
		}

		// 3. 트위터 찾기

		if strings.Contains(receivedMsg, "트윗") || strings.Contains(receivedMsg, "트위터") {

			log.Println("트위터 크롤링 시.")

			m := TwitterScrape(tweetenv)

			if len(m) == 0 {
				s.client.PostMessage(ev.Channel, "알 수 없는 에러가 발생했습니다. 다시 시도해 주세요.", slack.PostMessageParameters{})
			} else {
				for k, v := range m {

					attachment := slack.Attachment{

						Color: "#42c7d6",
						Title: k,
						Text:  v,
					}

					params := slack.PostMessageParameters{

						Attachments: []slack.Attachment{
							attachment,
						},
					}

					s.client.PostMessage(ev.Channel, "", params)
				}
			}
		}

		// 바. 깃허브 입력 시(최신유행 GO 오픈소스 찾기)
		/*
			if strings.Contains(receivedMsg, "깃허브") || strings.Contains(receivedMsg, "깃헙") {

				log.Println("깃허브 크롤링 시.")

				m := GoScrape()

				log.Println(m)

				for k, v := range m {

					title := strings.TrimPrefix(k, "/")
					title_link := "https://github.com" + strings.TrimSpace(k)

					attachment := slack.Attachment{

						Color:     "#f7b7ce",
						Title:     title,
						TitleLink: title_link,
						Text:      v,
					}

					params := slack.PostMessageParameters{

						Attachments: []slack.Attachment{
							attachment,
						},
					}

					s.client.PostMessage(ev.Channel, "", params)

				}

			}
		*/

		// 4. git 사용자이름 입력 시, 오늘의 깃허브 커밋여부 반환

		if strings.HasPrefix(receivedMsg, "git") {

			log.Println("깃 커밋 확인 시.")
			id := receivedMsg[strings.Index(receivedMsg, " ")+1:]
			strings.TrimSpace(id)

			// 사용자가 커밋을 하지 않았을 경우

			b, c := getGitCommit(id)

			if !b {

				if c == 1 {

					s.client.PostMessage(ev.Channel, "그런 유저가 없어요...", slack.PostMessageParameters{})

				} else {

					attachment := slack.Attachment{

						Color:     "#e20000",
						Title:     id + "님께서는 아직 커밋하신 적이 없습니다!",
						TitleLink: "https://github.com/" + id,
						Text:      "내용을 확인 해 주세요",
					}

					params := slack.PostMessageParameters{

						Attachments: []slack.Attachment{
							attachment,
						},
					}

					s.client.PostMessage(ev.Channel, "", params)

				}
			} else {

				attachment := slack.Attachment{

					Color:     "#e20000",
					Title:     id + "님께서는 오늘 " + fmt.Sprint(c) + "개의 커밋을 했습니다!",
					TitleLink: "https://github.com/" + id,
					Text:      "앞으로도 수고해 주세요",
				}

				params := slack.PostMessageParameters{

					Attachments: []slack.Attachment{
						attachment,
					},
				}

				s.client.PostMessage(ev.Channel, "", params)
			}
		}

		// 5. 근무자 입력 시, 현재 슬랙에 로그인 해 있는 상태인 사용자 반환

		if strings.Contains(receivedMsg, "근무자") {

			log.Println("현재 로그인 해 있는 사용자 확인 시")
			Users, _ := s.client.GetUsers()
			var logineduser []string

			for _, v := range Users {
				if v.Presence == "active" && v.IsBot == false {
					logineduser = append(logineduser, v.Name)
				}
			}

			attachment := slack.Attachment{

				Color: "#292963",
				Title: "현재 로그인 해 있는 사용자",
				Text:  strings.Join(logineduser, "\n"),
			}
			params := slack.PostMessageParameters{
				Attachments: []slack.Attachment{
					attachment,
				},
			}
			s.client.PostMessage(ev.Channel, "", params)
		}

		// 6. 도움 입력 시, 도움말을 전송

		if strings.Contains(receivedMsg, "도움") {
			log.Println("도움말!")

			attachment := slack.Attachment{

				Color: "#296346",
				Title: "봇 사용 커맨드",
				Text: `안녕하세요? IT봇입니다.
				IT봇 사용을 위해서 참고해주세요~
				1. @it_trend_go3 도움말 기능(개발중)
				2. @it_trend_go3 버튼 기능(개발중)
				2. 기사, 뉴스, 소식 키워드 입력 시 오늘의 IT 뉴스라인을 보실 수 있습니다.
				3. 오키, 옼희 입력 시 오키 주간 기술 트렌드를 보실 수 있습니다.
				4. 블로그 입력 시 엄선된 기술블로그들의 rss 피드를 얻어옵니다.
				5. 트위터, 트윗 입력 시 엄선된 트위터를 크롤링해 옵니다.
				6. git 사용자id(Ex - git hero0926) 입력 시 오늘의 커밋상황을 안내해 드립니다.
				7. 근무자 입력 시 현재 슬랙에 로그인 해 있는 사용자를 안내해 드립니다.`,
			}
			params := slack.PostMessageParameters{
				Attachments: []slack.Attachment{
					attachment,
				},
			}
			s.client.PostMessage(ev.Channel, "", params)

		}

		/* 테스트용 메서드~
		if strings.Contains(receivedMsg, "테스트") {
			params := slack.PostMessageParameters{
				Attachments: []slack.Attachment{},
			}

			s.client.PostMessage(userID, "디엠 테스트", params)
		}
		*/

		return nil

	}
	// 봇에게 멘션 했을 시

	if strings.HasPrefix(receivedMsg, fmt.Sprintf("<@%s> ", s.botID)) {

		log.Println("봇에게 멘션했을 시.")

		// 봇이 한 말이면 무시하자!
		if strings.Contains(ev.Msg.Username, "go돌이") {
			log.Println("봇이 한 대화라 무시 했어요.")
			return nil
		}

		// select 메뉴
		if strings.Contains(receivedMsg, "도움") {

			attachment := slack.Attachment{

				Text:       "무엇을 도와드릴까요? :newspaper: ",
				Color:      "#f9a41b",
				CallbackID: "news",
				Actions: []slack.AttachmentAction{

					{

						Name: actionSelect,
						Type: "select",

						Options: []slack.AttachmentActionOption{

							{
								Text:  "IT 기사 읽기",
								Value: "ITNews",
							},
							{
								Text:  "OKKY 읽기",
								Value: "OKKY",
							},
							{
								Text:  "TWITTER 읽기",
								Value: "TWITTER",
							},
							{
								Text:  "기술 블로그 읽기",
								Value: "BLOG",
							},
							{
								Text:  "도움말",
								Value: "HELP",
							},
						},
					},
				},
			}

			params := slack.PostMessageParameters{

				Attachments: []slack.Attachment{
					attachment,
				},
			}

			if _, _, err := s.client.PostMessage(ev.Channel, "", params); err != nil {
				return fmt.Errorf("failed to post message: %s", err)
			}

		} else if strings.Contains(receivedMsg, "버튼") {

			log.Println("버튼테스트")

			attachment := slack.Attachment{

				Text:       "버튼 테스트",
				Color:      "#f9a41b",
				CallbackID: "button",
				Actions: []slack.AttachmentAction{

					{
						Name:  "game",
						Text:  "개발",
						Type:  "button",
						Value: "chess",
					},
					{
						Name:  "game",
						Text:  "테스트",
						Type:  "button",
						Value: "chess2",
					},
					{
						Name:  "game",
						Text:  "누르지마세욧",
						Type:  "button",
						Value: "chess3",
						Style: "danger",
						Confirm: &slack.ConfirmationField{

							Title:       "ㅠㅠ",
							Text:        "서버와 연결 후 동작합니다",
							OkText:      "그래",
							DismissText: "아니",
						},
					},
				},
			}

			params := slack.PostMessageParameters{

				Attachments: []slack.Attachment{
					attachment,
				},
			}

			if _, _, err := s.client.PostMessage(ev.Channel, "", params); err != nil {
				return fmt.Errorf("failed to post message: %s", err)
			}

		} else {
			s.client.PostMessage(ev.Channel, "무엇을 도와드릴까요? 도움, 도움말 이라고 입력해보세요~", slack.PostMessageParameters{})
		}

	}

	log.Println("return nil")
	return nil
}

// 시간별로 채널에 메세지 보내기
func (s *SlackListener) PostByTime(env envConfig) {

	// 정확히 n시 0분 0초가 딱 정시 되는 순간 작동!
	for n := range GetHour().C {

		hour, _, _ := n.Clock()

		switch hour {
		case 12:
			attachment := slack.Attachment{

				Color:      "#a470e0",
				AuthorName: "점심알림",
				Title:      "점심 식사 하시러 갈 시간입니다!",
				Text:       "오늘도 맛있는 점심 되세요.",
			}
			params := slack.PostMessageParameters{
				Attachments: []slack.Attachment{
					attachment,
				},
			}
			s.client.PostMessage(env.ChannelID, "", params)

			// 시간별 커밋 알림봇 구현
		case 14:

			b, _ := getGitCommit("hero0926")
			if !b {
				attachment := slack.Attachment{

					Color:      "#635129",
					AuthorName: "Commit-bot",
					Title:      "아직 한 커밋이 없어요!",
				}
				params := slack.PostMessageParameters{
					Attachments: []slack.Attachment{
						attachment,
					},
				}

				//제가 새로 만든 유저에게 멘션을 보내는 메서드(풀 리퀘스트는 받아질 것인가?)
				//사용법 (보낼 채널, 보낼 텍스트, 보낼 유저(아이디), 파라미터)
				//그냥 쓰시려면 s.client.PostMessage(env.ChannelID, "<@유저아이디> ", params)
				//꼭 <> 를 넣어줘야 가더라고요...
				//s.client.PostMessageTo(env.ChannelID, "", "U6DKDJMPV", params)
				/*
					func (api *Client) PostMessageTo(channel, text string, id string, params PostMessageParameters) (string, string, error) {
						respChannel, respTimestamp, _, err := api.SendMessageContext(
							context.Background(),
							channel,
							MsgOptionText("<@"+id+"> "+text, params.EscapeText),
							MsgOptionAttachments(params.Attachments...),
							MsgOptionPostMessageParameters(params),
						)
						return respChannel, respTimestamp, err
					}
				*/

				//또는 디엠을 보내고 싶을때는 채널명에 유저ID를 쓰시면 됩니다.
				s.client.PostMessage("U6DKDJMPV", "", params)
			}
		case 15:
			b, _ := getGitCommit("hero0926")
			if !b {
				attachment := slack.Attachment{

					Color:      "#633f29",
					AuthorName: "Commit-bot",
					Title:      "아직도! 한 커밋이 없어요!",
				}
				params := slack.PostMessageParameters{
					Attachments: []slack.Attachment{
						attachment,
					},
				}
				s.client.PostMessage("U6DKDJMPV", "", params)
			}
		case 16:
			b, _ := getGitCommit("hero0926")
			if !b {
				attachment := slack.Attachment{

					Color:      "#632b29",
					AuthorName: "Commit-bot",
					Title:      "아직!!!!!!! 한개도 커밋이 없어요!",
				}
				params := slack.PostMessageParameters{
					Attachments: []slack.Attachment{
						attachment,
					},
				}
				s.client.PostMessage("U6DKDJMPV", "", params)
			}
		case 17:
			b, _ := getGitCommit("hero0926")
			if !b {
				attachment := slack.Attachment{

					Color:      "#680e0e",
					AuthorName: "Commit-bot",
					Title: `Commit-bot is watching your commit...
					PLZ commit soon...(아직도 안했다는 소리이다.)`,
				}
				params := slack.PostMessageParameters{
					Attachments: []slack.Attachment{
						attachment,
					},
				}
				s.client.PostMessage("U6DKDJMPV", "", params)
			}

		case 18:

			b, c := getGitCommit("hero0926")

			if !b {

				attachment := slack.Attachment{

					Color:      "#ff0033",
					AuthorName: "긴급 알림",
					Title:      "퇴근 할 시간인데도 커밋을 하지 않았습니다!",
					Text:       "뭔가 하고 가시던지 집에 가서 해보세요!",
				}
				params := slack.PostMessageParameters{
					Attachments: []slack.Attachment{
						attachment,
					},
				}

				s.client.PostMessage("U6DKDJMPV", "", params)

			} else {

				attachment := slack.Attachment{

					Color:      "#ff0033",
					AuthorName: "수고의 알림",
					Title:      "퇴근 할 시간입니다!",
					Text: `오늘도 수고하셨어요. ` +
						"오늘은" + fmt.Sprint(c) + "개의 커밋을 하였습니다.",
				}
				params := slack.PostMessageParameters{
					Attachments: []slack.Attachment{
						attachment,
					},
				}

				s.client.PostMessage("U6DKDJMPV", "", params)
			}

			attachment := slack.Attachment{

				Color:      "#ff0033",
				AuthorName: "퇴근알림",
				Title:      "퇴근 할 시간입니다! ",
				Text:       "오늘도 수고하셨어요.",
			}
			params := slack.PostMessageParameters{
				Attachments: []slack.Attachment{
					attachment,
				},
			}

			s.client.PostMessage(env.ChannelID, "", params)

			// 야근봇 구현
			// 퇴근 후 일정시간 자동 백업 등을 수행할 수 있을 것 같음...
		case 19, 20, 21:

			Users, _ := s.client.GetUsers()
			var logineduser []string

			for _, v := range Users {
				if v.Presence == "active" && v.IsBot == false {
					logineduser = append(logineduser, v.Name)
				}
			}

			attachment := slack.Attachment{

				Color:      "#63294e",
				Pretext:    "아직 불철주야 일하고 계신 분",
				AuthorName: "현재 근무자",
				Title:      strings.Join(logineduser, "\n"),
				Text:       "님께서" + fmt.Sprint(hour) + "시까지 수고해주시고 계십니다.",
			}
			params := slack.PostMessageParameters{
				Attachments: []slack.Attachment{
					attachment,
				},
			}
			s.client.PostMessage(env.ChannelID, "", params)

		}
	}
}

// 정시 얻기

/*
이걸 활용해서 매일 n시에 기사 크롤링을 해온 후 저장해 뒀다 선별해서 보여줄 수도 있고
이걸 활용해서 매일 n시에 사용자의 작업을 확인한 후 메시지를 보내 줄 수도 있을 것 같음
또는 주변 맛집을 찾아다가 점심시간에 투표 포스팅을 할 수도 있음
*/

func GetHour() *time.Ticker {
	c := make(chan time.Time, 1)
	t := &time.Ticker{C: c}
	go func() {
		for {
			n := time.Now()
			if n.Second() == 0 && n.Minute() == 0 {
				c <- n
			}
			time.Sleep(time.Second)
		}
	}()
	return t
}