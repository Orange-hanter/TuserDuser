import datetime
import re
import subprocess
import uuid

import telebot
from telebot import types
from db import init_db

from config import token



bot = telebot.TeleBot(token)

keyboard = telebot.types.ReplyKeyboardMarkup(row_width=1, resize_keyboard=True)
events_today = types.KeyboardButton('Эвенты на сегодня')
events_tommorow = types.KeyboardButton('Эвенты на завтра')
events_onWeek = types.KeyboardButton('Эвенты на неделе')
add_event = types.KeyboardButton('Добавить эвент')
#keyboard1.row(events_today, events_tommorow,events_onWeek)
keyboard.add(events_today, events_tommorow,events_onWeek,add_event)


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
    #print(message.text)
    if message.text == 'Эвенты на сегодня':
        
        bot.send_message(message.chat.id, "сегодня ничего не проиходит", reply_markup=markup)
        
    elif message.text == 'Эвенты на завтра':
        
        bot.send_message(message.chat.id, "завтра ничего не проиходит", reply_markup=markup)
        
        
    elif message.text == 'Эвенты на неделе':
        
        bot.send_message(message.chat.id, "на неделе ничего не проиходит", reply_markup=markup)
        
    elif message.text == 'Добавить эвент':
        
        bot.send_message(message.chat.id, "Введите описание", reply_markup=markup)
        bot.register_next_step_handler(msg, process_descr_step)
        



def process_descr_step(message):
    try:
        chat_id = message.chat.id
        description = message.text
        msg = bot.reply_to(message, 'Описание')
        bot.register_next_step_handler(msg, process_age_step)
    except Exception as e:
        bot.reply_to(message, 'oooops')

        

def process_date_step(message):
    try:
        chat_id = message.chat.id
        description = message.text
        msg = bot.reply_to(message, 'Описание')
        bot.register_next_step_handler(msg, process_age_step)
    except Exception as e:
        bot.reply_to(message, 'oooops')


if __name__ == '__main__': 
    init_db()
    bot.polling()
