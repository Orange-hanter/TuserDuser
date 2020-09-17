import datetime
from dateutil.parser import parse
import re
import subprocess
import uuid
import telebot
from telebot import types
from db import init_db, \
                get_events_today_db, \
                add_event_db, \
                get_events_by_day_db, \
                get_events_by_period_db
from config import token


class Event:
    def __init__(self, description):
        self.description = description
        self.date = None


bot = telebot.TeleBot(token)
event_dict = {}
keyboard = None


def tomorrow_date():
    return datetime.date.today() + datetime.timedelta(days=1)


def init_bot():
    global keyboard
    keyboard = telebot.types.ReplyKeyboardMarkup(row_width=2, resize_keyboard=True)

    events_today = types.KeyboardButton('События сегодня')
    events_tomorrow = types.KeyboardButton('События завтра')
    events_on_week = types.KeyboardButton('События на неделе')
    add_event = types.KeyboardButton('Добавить эвент')

    # keyboard1.row(events_today, events_tomorrow,events_onWeek)
    keyboard.add(events_today, events_tomorrow, events_on_week, add_event)


def add_new_event_proc(message):
    try:
        event = Event(str(message.text))
        event_dict[message.chat.id] = event
        msg = bot.reply_to(message, 'Введите дату (ДД/ММ/ГГГГ)')
        bot.register_next_step_handler(msg, process_date_step)
    except Exception as e:
        bot.reply_to(message, 'Oooops: ' + str(e))


def process_date_step(message):
    try:
        chat_id = message.chat.id
        event = event_dict[chat_id]
        event.date = parse(str(message.text)).date()
        print(parse(str(message.text)))
        bot.send_message(chat_id, 'Хорошо!\nСобытие: ' + event.description + '\nВремя: ' + str(event.date))
        add_event_db(event.description, event.date)
    except Exception as e:
        print("Exception: " + str(e))
        bot.reply_to(message, 'Введите дату в формате')     # Что за странная обработка ошибки?


def render_events(events):
    rendered_text = ''
    for event in events:
        rendered_text += f"Когда: {event[2]}\nЧто: {event[1]}\n\n"
    return rendered_text


@bot.message_handler(commands=['start', 'help'])
def start_message(message):
    """
    Функция отвечает на команды 'start', 'help'
    :param message:
    :return:
    """
    button_name = {"Start": "[События сегодня]",
                   }
    bot_description = f"Привет, я бот котрый напомнит тебе о мероприятиях Бреста. " \
                      f"Если хочешь узнать что сегодня будет интересного нажми кнопку {button_name['Start']}\n"
    bot.send_message(message.chat.id, bot_description, reply_markup=keyboard)


@bot.message_handler(content_types=['text'])
def command_handler(message):
    """
    Основная функция обработчик
    :param message:
    :return:
    """
    markup = types.ReplyKeyboardMarkup()
    user_id = message.chat.id
    request = message.text
    print("INPUT MESSAGE: " + message.text)

    if request == 'События сегодня':
        events = get_events_today_db()
        response = render_events(events) if not events == [] else "Сегодня ничего не проиходит"
        bot.send_message(user_id, response, reply_markup=markup)

    elif request == 'События завтра':
        events = get_events_by_day_db(tomorrow_date())
        response = render_events(events) if not events == [] else "Завтра ничего не проиходит"
        bot.send_message(user_id, response, reply_markup=markup)

    elif request == 'События на неделе':
        events = get_events_by_period_db(tomorrow_date(), datetime.timedelta(days=7))
        response = render_events(events) if not events == [] else "На неделе ничего не проиходит"
        bot.send_message(user_id, response, reply_markup=markup)

    elif request == 'Добавить эвент':
        bot.send_message(user_id, "Что за мероприятие?", reply_markup=markup)
        bot.register_next_step_handler(message, add_new_event_proc)


if __name__ == '__main__':
    init_db()
    init_bot()
    bot.polling()
