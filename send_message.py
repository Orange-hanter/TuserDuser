import logging
import sys

from telebot import types

from main import bot




def send_message(id, text):  # Функция отправляет сообщения в чат пользователю

    bot.send_message(id, text)


if __name__ == "__main__":
    send_message(sys.argv[1], sys.argv[2])
