import datetime
from dateutil.parser import parse
import re
import subprocess
import uuid

import telebot
from telebot import types
from db import init_db, get_events_today_db, add_event_db

from config import token

bot = telebot.TeleBot(token)

keyboard = telebot.types.ReplyKeyboardMarkup(row_width=2, resize_keyboard=True)

events_today = types.KeyboardButton('Эвенты на сегодня')
events_tomorrow = types.KeyboardButton('Эвенты на завтра')
events_onWeek = types.KeyboardButton('Эвенты на неделе')
add_event = types.KeyboardButton('Добавить эвент')
# keyboard1.row(events_today, events_tomorrow,events_onWeek)
keyboard.add(events_today, events_tomorrow, events_onWeek, add_event)


event_dict = {}

class Event:
    def __init__(self, description):
        self.description = description
        self.date = None


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

        bot.register_next_step_handler(message, process_descr_step)
        



def process_descr_step(message):
    try:
        
        chat_id = message.chat.id
        description = str(message.text)
        event = Event(description)
        event_dict[chat_id] = event
        msg = bot.reply_to(message, 'Введите дату')
        bot.register_next_step_handler(msg, process_date_step)
    except Exception as e:
        bot.reply_to(message, 'oooops')

        

def process_date_step(message):
    try:
        chat_id = message.chat.id
        event = event_dict[chat_id]
        date = message.text
        event.date  = parse(str(date))

        bot.send_message(chat_id, 'Хорошо \n' + event.description + '\n когда:' + str(event.date))
        add_event_db(event.description,event.date)
    except Exception as e:
        print(e)
        bot.reply_to(message, 'введите дату в формате')



if __name__ == '__main__':
    init_db()
    bot.polling()
