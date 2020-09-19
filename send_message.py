import logging
import sys

from telebot import types

import telebot
from telebot import types

from config import token

bot = telebot.TeleBot(token)


def send_message(id, text):  # Функция отправляет сообщения в чат пользователю
    bot.send_message(id, 'Напоминание:\n'+text)



if __name__ == "__main__":
    send_message(sys.argv[1], str(sys.argv[2]))


