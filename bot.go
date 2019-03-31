package main

// сюда писать код

import (
	"context"
	"net/http"
	"os"
	"strconv"
	"strings"

	tgbotapi "gopkg.in/telegram-bot-api.v4"
)

var (
	// @BotFather в телеграме даст вам это
	BotToken = "XXX"

	// урл выдаст вам игрок или хероку
	WebhookURL = "https://525f2cb5.ngrok.io"
)

type User struct {
	Name   string
	ChatID int64
}

type Task struct {
	ID          int
	Description string
	Owner       User
	Assignee    User
}

var taskID = 0

func startTaskBot(ctx context.Context) error {
	// сюда пишите ваш код
	tasks := make([]*Task, 0)
	bot, err := tgbotapi.NewBotAPI(BotToken)
	if err != nil {
		return err
	}
	_, err = bot.SetWebhook(tgbotapi.NewWebhook(WebhookURL))
	if err != nil {
		return err
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}
	go http.ListenAndServe(":"+port, nil)
	updates := bot.ListenForWebhook("/")
	for update := range updates {
		text := update.Message.Text
		user := update.Message.From.UserName
		if text == "/tasks" {
			bot.Send(tgbotapi.NewMessage(
				update.Message.Chat.ID,
				tasksString(getTasks(tasks, user), user),
			))
		} else if strings.Contains(text, "/new") {
			taskDescription := text[len("/new "):]
			bot.Send(tgbotapi.NewMessage(
				update.Message.Chat.ID,
				addTask(&tasks, user, update.Message.Chat.ID, taskDescription),
			))
		} else if strings.Contains(text, "/assign_") {
			taskID, err := strconv.Atoi(text[len("/assign_"):])
			if err != nil {
				bot.Send(tgbotapi.NewMessage(
					update.Message.Chat.ID,
					"Неверный идентификатор задачи",
				))
			} else {
				for chatID, message := range assignTaskByID(taskID, &tasks, user, update.Message.Chat.ID) {
					bot.Send(tgbotapi.NewMessage(
						chatID,
						message,
					))
				}
			}

		} else if strings.Contains(text, "/unassign_") {
			taskID, err := strconv.Atoi(text[len("/unassign_"):])
			if err != nil {
				bot.Send(tgbotapi.NewMessage(
					update.Message.Chat.ID,
					"Неверный идентификатор задачи",
				))
			} else {
				for chatID, message := range unassignTaskByID(taskID, &tasks, user, update.Message.Chat.ID) {
					bot.Send(tgbotapi.NewMessage(
						chatID,
						message,
					))
				}
			}
		} else if strings.Contains(text, "/resolve_") {
			taskID, err := strconv.Atoi(text[len("/resolve_"):])
			if err != nil {
				bot.Send(tgbotapi.NewMessage(
					update.Message.Chat.ID,
					"Неверный идентификатор задачи",
				))
			} else {
				for chatID, message := range resolveTaskByID(taskID, &tasks, user, update.Message.Chat.ID) {
					bot.Send(tgbotapi.NewMessage(
						chatID,
						message,
					))
				}
			}
		} else if text == "/my" {
			bot.Send(tgbotapi.NewMessage(
				update.Message.Chat.ID,
				getMyTasksInfo(tasks, user, "assignee"),
			))
		} else if text == "/owner" {
			bot.Send(tgbotapi.NewMessage(
				update.Message.Chat.ID,
				getMyTasksInfo(tasks, user, "owner"),
			))
		} else {
			bot.Send(tgbotapi.NewMessage(
				update.Message.Chat.ID,
				"Неизвестная команда",
			))
		}

	}
	return nil
}

func tasksString(tasks []Task, user string) string {
	if len(tasks) == 0 {
		return "Нет задач"
	}
	result := ""
	for _, task := range tasks {
		result += strconv.Itoa(task.ID) + ". " + task.Description + " by @" + task.Owner.Name + "\n"
		if task.Assignee.Name != "" {
			if user == task.Assignee.Name {
				taskID := strconv.Itoa(task.ID)
				result += "assignee: я\n/unassign_" + taskID + " /resolve_" + taskID + "\n"
			} else {
				result += "assignee: @" + task.Assignee.Name + "\n"
			}
		} else {
			result += "/assign_" + strconv.Itoa(task.ID) + "\n"
		}
		result += "\n"
	}
	return result[:len(result)-2]
}

func getTasks(tasks []*Task, user string) []Task {
	ownerTasks := make([]Task, 0)
	for _, task := range tasks {
		if task != nil {
			ownerTasks = append(ownerTasks, *task)
		}
	}
	return ownerTasks
}

func addTask(tasks *[]*Task, owner string, ownerChatID int64, taskDescription string) string {
	taskID++
	newTask := Task{ID: taskID, Description: taskDescription, Owner: User{Name: owner, ChatID: ownerChatID}}
	*tasks = append(*tasks, &newTask)
	return "Задача \"" + taskDescription + "\" создана, id=" + strconv.Itoa(newTask.ID)
}

func assignTaskByID(taskID int, tasks *[]*Task, assignee string, assigneeChatID int64) map[int64]string {
	result := make(map[int64]string, 2)
	for _, task := range *tasks {
		if task.ID == taskID {
			if task.Assignee.Name != "" {
				result[task.Assignee.ChatID] = "Задача \"" + task.Description + "\" назначена на @" + assignee
			} else {
				result[task.Owner.ChatID] = "Задача \"" + task.Description + "\" назначена на @" + assignee
			}
			task.Assignee = User{Name: assignee, ChatID: assigneeChatID}
			result[assigneeChatID] = "Задача \"" + task.Description + "\" назначена на вас"
		}
	}
	return result
}

func unassignTaskByID(taskID int, tasks *[]*Task, user string, userChatID int64) map[int64]string {
	result := make(map[int64]string, 2)
	for _, task := range *tasks {
		if task.ID == taskID {
			if task.Assignee.Name != user {
				result[userChatID] = "Задача не на вас"
			} else {
				result[userChatID] = "Принято"
				task.Assignee = User{Name: "", ChatID: 0}
				result[task.Owner.ChatID] = "Задача \"" + task.Description + "\" осталась без исполнителя"
			}
		}
	}
	return result
}

func resolveTaskByID(taskID int, tasks *[]*Task, user string, userChatID int64) map[int64]string {
	result := make(map[int64]string, 2)
	for idx, task := range *tasks {
		if task.ID == taskID {
			if task.Assignee.Name != user {
				result[userChatID] = "Задача не на вас"
			} else {
				result[userChatID] = "Задача \"" + task.Description + "\" выполнена"
				result[task.Owner.ChatID] = "Задача \"" + task.Description + "\" выполнена @" + user
				*tasks = append((*tasks)[:idx], (*tasks)[idx+1:]...)
			}
		}
	}
	return result
}

func getMyTasksInfo(tasks []*Task, user string, role string) string {
	result := ""
	for _, task := range tasks {
		if task != nil {
			if role == "assignee" && (*task).Assignee.Name == user {
				taskID := strconv.Itoa((*task).ID)
				result += taskID + ". " + (*task).Description + " by @" + (*task).Owner.Name + "\n/unassign_" + taskID + " /resolve_" + taskID + "\n"
			} else if role == "owner" && (*task).Owner.Name == user {
				taskID := strconv.Itoa((*task).ID)
				result += taskID + ". " + (*task).Description + " by @" + (*task).Owner.Name + "\n/assign_" + taskID + "\n"
			}

		}
	}
	return result[:len(result)-1]
}

func main() {
	err := startTaskBot(context.Background())
	if err != nil {
		println(err.Error)
	}
}
