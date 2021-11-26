import sys
import telebot

from Service.config import token

bot = telebot.TeleBot(token)


def send_message(id, text):  # Функция отправляет сообщения в чат пользователю
    bot.send_message(id, 'Напоминание:\n' + text)


if __name__ == "__main__":
    send_message(sys.argv[1], str(sys.argv[2]))
