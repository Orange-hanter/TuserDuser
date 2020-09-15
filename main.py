import datetime
import re
import subprocess
import uuid

import telebot
from telebot import types
from db import init_db, get_events_today_db

from config import token

bot = telebot.TeleBot(token)

keyboard = telebot.types.ReplyKeyboardMarkup(row_width=2, resize_keyboard=True)
events_today = types.KeyboardButton('Эвенты на сегодня')
events_tomorrow = types.KeyboardButton('Эвенты на завтра')
events_onWeek = types.KeyboardButton('Эвенты на неделе')
add_event = types.KeyboardButton('Добавить эвент')
# keyboard1.row(events_today, events_tomorrow,events_onWeek)
keyboard.add(events_today, events_tomorrow, events_onWeek, add_event)


@bot.message_handler(commands=['start', 'help'])  # Функция отвечает на команды 'start', 'help'
def start_message(message):
    bot.send_message(message.chat.id,
                     f"Привет, я бот котрый напомнит тебе о мероприятиях Бреста. \n", reply_markup=keyboard)


@bot.message_handler(content_types=['text'])
def in_text(message):
    markup = types.ReplyKeyboardMarkup()
    print(message.text)

    if message.text == 'Эвенты на сегодня':
        events = get_events_today_db()
        if len(events) != 0:
            bot.send_message(message.chat.id, str(events), reply_markup=markup)
        else:
            bot.send_message(message.chat.id, "сегодня ничего не проиходит", reply_markup=markup)

    elif message.text == 'Эвенты на завтра':
        bot.send_message(message.chat.id, "завтра ничего не проиходит", reply_markup=markup)

    elif message.text == 'Эвенты на неделе':
        bot.send_message(message.chat.id, "на неделе ничего не проиходит", reply_markup=markup)

    elif message.text == 'Добавить эвент':
        bot.send_message(message.chat.id, "Что за мероприятие?", reply_markup=markup)
    # Усталь :(


if __name__ == '__main__':
    init_db()
    bot.polling()
