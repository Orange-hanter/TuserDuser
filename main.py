import datetime
from datetime import date
import os
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
    get_events_by_period_db, add_to_db_tasklist, add_user, get_user_role,get_event_by_id
from config import token


class Event:
    def __init__(self, description):
        self.description = description
        self.date = None
        self.url = None


bot = telebot.TeleBot(token)
event_dict = {}
keyboard = None

# для связи id эвента и id сообщения в будущем скорее всего будет в бд
event_id_and_message_id = {}


def tomorrow_date():
    return datetime.date.today() + datetime.timedelta(days=1)


def init_bot():
    global keyboard
    global admin_keyboard
    keyboard = telebot.types.ReplyKeyboardMarkup(row_width=2, resize_keyboard=True)
    admin_keyboard = telebot.types.ReplyKeyboardMarkup(row_width=2, resize_keyboard=True)

    events_today = types.KeyboardButton('События сегодня')
    events_tomorrow = types.KeyboardButton('События завтра')
    events_on_week = types.KeyboardButton('События на неделе')

    add_event = types.KeyboardButton('Добавить эвент')

    # keyboard1.row(events_today, events_tomorrow,events_onWeek)
    keyboard.add(events_today, events_tomorrow, events_on_week, add_event)
    admin_keyboard.add(events_today, events_tomorrow, events_on_week, add_event)


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


    except Exception as e:
        print("Exception: " + str(e))
        bot.reply_to(message, 'Введите дату в формате')  # Что за странная обработка ошибки?


def add_new_event_url(message):
    try:
        chat_id = message.chat.id
        event = event_dict[chat_id]
        event.url = str(message.text)
        add_event_db(event.description, event.date, event.url)
        bot.send_message(chat_id, 'Хорошо!\nСобытие: ' + event.description + '\nВремя: ' + str(event.date))
    except Exception as e:
        bot.reply_to(message, 'Oooops: ' + str(e))


def render_events(events):
    rendered_text = []
    for event in events:
        rendered_text.append(f"Когда: {event[2]}\nЧто: {event[1]}\n\n")
    return rendered_text


def send_messgage_with_reminder_and_url(messgage, user_id, request, event_url, event_id):
    keyboard = types.InlineKeyboardMarkup()
    keyboard.add(types.InlineKeyboardButton(text='Напомнить за 15 минут', callback_data=' за 15 минут'))
    keyboard.add(types.InlineKeyboardButton(text='Напомнить за час', callback_data=' за час'))

    if request != 'Эвенты на сегодня':
        keyboard.add(types.InlineKeyboardButton(text='Напомнить завтра', callback_data=' завтра'))

    url_button = types.InlineKeyboardButton(text="Ссылка", url=event_url)
    keyboard.add(url_button)

    message_id = bot.send_message(user_id, messgage, reply_markup=keyboard).message_id
    # event_id_and_message_id[event_id].append(message_id)
    event_id_and_message_id.update({message_id: event_id})


def process_messages(events, user_id, request):
    for event in events:
        # print(event)
        event_id = event[0]
        event_url = event[3]
        # event_id_and_message_id.update({event_id: []})

        messgage = f"Когда: {event[2]}\nЧто: {event[1]}\n\n"

        send_messgage_with_reminder_and_url(messgage, user_id, request, event_url, event_id)


def get_datetime(call):
    text = call.message.text
    match = re.search(r'\d{4}-\d{2}-\d{2}', text)
    date = datetime.datetime.strptime(match.group(), '%Y-%m-%d').date()
    return date




def add_task(id, text, date_time):  # Функция создают задачу в AT и добавляет ее в бд
    uid = uuid.uuid4()

    # cmd = f"""echo "send_message.py {id}  \'{text}\' {uid} '' " | at {date_time}"""
    cmd = ['python3', 'send_message.py', str(id), str(text), '| at', str(date_time)]
    out = subprocess.Popen(cmd, shell=False, stdout=subprocess.PIPE, stderr=subprocess.STDOUT)
    stdout = str(out.communicate())
    # number = re.search('job(.+?) at', stdout).group(1)
    add_to_db_tasklist(id, date_time, text, uid)


@bot.callback_query_handler(func=lambda call: True)  # Реакция на кнопки
def callback(call):
    if call.data == ' за 15 минут':
        event_id = event_id_and_message_id[call.message.message_id]

        date_time = parse(get_event_by_id(event_id)[0][2])
        print(date_time)
        date_time = date_time + datetime.timedelta(minutes=1)
        date_time = date_time.strftime("%H:%M %m%d%y")
        print(date_time)
        add_task(call.from_user.id, call.message.text, date_time)

    # if call.data == ' за час':
    # if call.data == ' завтра':


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

    add_user(message.chat.id, 'user')


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
        print(events)

        render_events(events) if not events == [] else "Сегодня ничего не проиходит"
        # print(response)
        process_messages(events, user_id, request)



    elif request == 'События  завтра':
        events = get_events_by_day_db(tomorrow_date())
        response = render_events(events) if not events == [] else "Завтра ничего не проиходит"
        bot.send_message(user_id, response, reply_markup=markup)

    elif request == 'События на неделе':
        events = get_events_by_period_db(tomorrow_date(), datetime.timedelta(days=7))
        response = render_events(events) if not events == [] else "На неделе ничего не проиходит"
        bot.send_message(user_id, response, reply_markup=markup)

    elif request == 'Добавить эвент':
        # Пока оставлю так
        bot.send_message(user_id, "Что за мероприятие?", reply_markup=markup)
        bot.register_next_step_handler(message, add_new_event_proc)
        # print(get_user_role(user_id)[0][0]=='admin')

        role = get_user_role(user_id)[0][0]
        # if role == 'admin' or role == 'client:
        #     bot.send_message(user_id, "Что за мероприятие?", reply_markup=markup)
        #     bot.register_next_step_handler(message, add_new_event_proc)
        # else:
        #     bot.send_message(user_id, 'Вы не админ', reply_markup=markup)


if __name__ == '__main__':
    init_db()
    init_bot()
    bot.polling()
