import datetime
import logging
import os
import re
import subprocess
import time

import telebot
import telebot_calendar
from dateutil.parser import parse
from telebot import types
from telebot.types import ReplyKeyboardRemove, CallbackQuery
from telebot_calendar import CallbackData

from Service.config import token

logging.basicConfig(filename="../history_work.log", level=logging.INFO)


# telebot.logger.setLevel(logging.DEBUG) # Outputs debug messages to console.

class Event:
    def __init__(self, description):
        self.description = description
        self.date = None
        self.url = None
        self.time = None


bot = telebot.TeleBot(token)

event_dict = {}
keyboard = None

# Creates a unique calendar
calendar_1 = CallbackData("calendar_1", "action", "year", "month", "day")


def tomorrow_date():
    return datetime.date.today() + datetime.timedelta(days=1)


def init_bot():
    global keyboard
    global admin_keyboard
    global client_keyboard
    keyboard = telebot.types.ReplyKeyboardMarkup(row_width=2, resize_keyboard=True)
    admin_keyboard = telebot.types.ReplyKeyboardMarkup(row_width=2, resize_keyboard=True)
    client_keyboard = telebot.types.ReplyKeyboardMarkup(row_width=2, resize_keyboard=True)

    events_today = types.KeyboardButton('События сегодня')
    events_tomorrow = types.KeyboardButton('События завтра')
    events_on_week = types.KeyboardButton('События на неделе')
    add_event = types.KeyboardButton('Добавить эвент')
    add_client = types.KeyboardButton('Добавить клиента')
    get_file_version = types.KeyboardButton('Версия')
    # keyboard1.row(events_today, events_tomorrow,events_on_week)
    keyboard.add(events_today, events_tomorrow, events_on_week)
    client_keyboard.add(events_today, events_tomorrow, events_on_week, add_event)
    admin_keyboard.add(events_today, events_tomorrow, events_on_week, add_event, add_client, get_file_version)


def send_message_to_admins():
    from Service.config import admins
    for admin_id in admins:
        bot.send_message(int(admin_id), 'Бот запущен ' + get_version())


def add_new_event_proc(message):
    try:
        description = str(message.text)
        if description == 'Отмена':
            cancel_adding_event(message.chat.id)
            return

        event = Event(description)
        event_dict[message.chat.id] = event
        # msg = bot.reply_to(message, 'Введите дату в формате  (ДД/ММ/ГГГГ)')
        now = datetime.datetime.now()  # Get the current date
        bot.reply_to(
            message,
            "Выбери дату",
            reply_markup=telebot_calendar.create_calendar(
                name=calendar_1.prefix,
                year=now.year,
                month=now.month,  # Specify the NAME of your calendar
            ),
        )
        # bot.register_next_step_handler(msg, process_date_step)
    except Exception as e:
        print(str(e))

        # msg = bot.reply_to(message, 'Введите описание')
        # bot.register_next_step_handler(msg, process_date_step)


def process_date_step(message):
    try:
        date = message.text
        if date == 'Отмена':
            cancel_adding_event(message.chat.id)
            return
        chat_id = message.chat.id
        event = event_dict[chat_id]

        date = parse(str(date)).date()
        if date < datetime.date.today():
            # msg = bot.reply_to(message, 'Это прошлое. Введите дату в формате (ДД/ММ/ГГГГ)')
            now = datetime.datetime.now()  # Get the current date
            bot.reply_to(
                message,
                "Это прошлое. Выбери дату",
                reply_markup=telebot_calendar.create_calendar(
                    name=calendar_1.prefix,
                    year=now.year,
                    month=now.month,
                ),
            )

            return

        event.date = date
        markup = types.ReplyKeyboardMarkup(one_time_keyboard=True, resize_keyboard=True)
        markup.add('Отмена')
        msg = bot.reply_to(message, 'Введите время (ЧЧ:ММ)', reply_markup=markup)
        bot.register_next_step_handler(msg, process_time_step)

    except Exception as e:
        # msg = bot.reply_to(message, 'Введите дату в формате (ДД/ММ/ГГГГ)')
        # bot.register_next_step_handler(msg, process_date_step)
        print(str(e))


def process_time_step(message):
    try:
        time = message.text
        if time == 'Отмена':
            cancel_adding_event(message.chat.id)
            return
        chat_id = message.chat.id
        event = event_dict[chat_id]
        time = re.search(r'\d{2}[:|/|-|.| ]\d{2}', time)
        if time == None:
            msg = bot.reply_to(message, 'Введите время в формате (ЧЧ:ММ)')
            bot.register_next_step_handler(msg, process_date_step)

        event.time = re.sub(r'[:|/|-|.| ]', ':', time.group())
        markup = types.ReplyKeyboardMarkup(one_time_keyboard=True, resize_keyboard=True)
        markup.add('Нет ссылки', 'Отмена')
        msg = bot.reply_to(message, 'Введите ссылку', reply_markup=markup)
        bot.register_next_step_handler(msg, add_new_event_url)

    except Exception as e:
        print("Exception: " + str(e))
        bot.reply_to(message, 'Введите время в формате (ЧЧ/ММ)')


def add_new_event_url(message):
    try:
        url = str(message.text)
        if url == 'Отмена':
            cancel_adding_event(message.chat.id)
            return
        chat_id = message.chat.id
        event = event_dict[chat_id]

        if url == 'Нет ссылки':
            url = ''
        event.url = url

        markup = types.ReplyKeyboardMarkup(one_time_keyboard=True, resize_keyboard=True)
        markup.add('Нет изображения', 'Отмена')
        msg = bot.reply_to(message, 'Добавьте изображение', reply_markup=markup)
        bot.register_next_step_handler(msg, add_new_event_image)

    except Exception as e:
        # bot.reply_to(message, 'Oooops: ' + str(e))
        print(e)


def add_new_event_image(message):
    try:
        if message.text == 'Отмена':
            cancel_adding_event(message.chat.id)
            return

        chat_id = message.chat.id
        event = event_dict[chat_id]

        image_id = None
        if message.content_type == 'photo':
            image_id = message.photo[0].file_id

        date_time_event = str(event.time) + ' ' + str(event.date)
        creator_id = chat_id
        add_event_db(event.description, event.date, event.time, event.url, image_id, creator_id)
        keyboard_keyboard = get_keyboard_by_id(chat_id)
        bot.send_message(chat_id, 'Хорошо!\nСобытие: ' + event.description + '\nВремя: ' + date_time_event,
                         reply_markup=keyboard_keyboard)
        del event_dict[chat_id]
    except Exception as e:
        print(e)
        # bot.reply_to(message, 'Oooops: ' + str(e))


def add_new_client(message):
    chat_id = message.chat.id
    keyboard_keyboard = get_keyboard_by_id(chat_id)
    if message.text == 'Отмена':
        cancel_adding_event(message.chat.id)
        return

    if message.forward_from != None:
        add_user(message.forward_from.id, 'client')
    else:
        add_user(chat_id, 'client')

    bot.send_message(chat_id, 'Клиент добавлен', reply_markup=keyboard_keyboard)


def cancel_adding_event(chat_id):
    keyboard_keyboard = get_keyboard_by_id(chat_id)
    bot.send_message(chat_id, 'Хорошо, добавление отменено.', reply_markup=keyboard_keyboard)
    if chat_id in event_dict:
        del event_dict[chat_id]


def send_messgage_with_reminder(messgage, user_id, request, event_url, event_id, date_time, image_id,
                                cancel_button=False):
    event = get_event_by_id(event_id)[0]
    # print(str(event[3]))
    event_date = parse(' '.join([str(event[2]), str(event[3])]))

    keyboard = types.InlineKeyboardMarkup()

    now = datetime.datetime.now()  # Now
    duration = event_date - now  # For build-in functions
    duration_in_s = duration.total_seconds()
    minutes = divmod(duration_in_s, 60)[0]
    if minutes < 0:
        return
    if minutes > 15:
        keyboard.add(types.InlineKeyboardButton(text='Напомнить за 15 минут', callback_data=' за 15 минут'))
    if minutes > 60:
        keyboard.add(types.InlineKeyboardButton(text='Напомнить за час', callback_data=' за час'))
    if event_url != '':
        url_button = types.InlineKeyboardButton(text="Ссылка", url=event_url)
        keyboard.add(url_button)
    if image_id != 'None':
        bot.send_photo(user_id, photo=image_id)
    if cancel_button:
        keyboard.add(types.InlineKeyboardButton(text='Отменить эвент', callback_data=' Отменить эвент'))
    # print(api_ret)
    message_id = bot.send_message(user_id, messgage, reply_markup=keyboard).message_id

    # event_id_and_message_id[event_id].append(message_id)
    # event_id_and_message_id.update({message_id: event_id})
    put_event_id_and_message_id(message_id, event_id)


def process_messages(events, user_id, request):
    for event in events:
        if not event:
            continue
        # print(event)
        event_id = event[0]
        event_url = event[4]
        date = event[2]
        image_id = event[5]
        creator_id = event[6]

        if image_id.startswith('DB'):
            image_id = open(image_id, "rb")

        # event_id_and_message_id.update({event_id: []})
        date_moved = date[-2:] + date[4:8] + date[:4]
        messgage = f"Когда: {date_moved}, в {event[3]}\nЧто: {event[1]}\n\n"
        cancel_button = False

        if get_user_role(user_id)[0][0] == 'admin' or creator_id == user_id:
            cancel_button = True
        send_messgage_with_reminder(messgage, user_id, request, event_url, event_id, date, image_id, cancel_button)


def get_datetime(call):
    text = call.message.text
    match = re.search(r'\d{4}-\d{2}-\d{2}', text)
    date = datetime.datetime.strptime(match.group(), '%Y-%m-%d').date()
    return date


def get_keyboard_by_id(user_id):
    markup_keyboard = None
    print(get_user_role(user_id)[0])
    if get_user_role(user_id)[0][0] == 'admin':
        markup_keyboard = admin_keyboard

    elif get_user_role(user_id)[0][0] == 'client':
        markup_keyboard = client_keyboard

    elif get_user_role(user_id)[0][0] == 'user':
        markup_keyboard = keyboard

    return markup_keyboard


def add_task(user_id, text, date_time, event_id):  # Функция создают задачу в AT и добавляет ее в бд
    cmd = f"python3 send_message.py {str(user_id)} '{text}'"
    # print(cmd)
    out = subprocess.check_output(["at", str(date_time)], input=cmd.encode(), stderr=subprocess.STDOUT)
    # stdout = str(out.communicate())
    number = int(re.search('job(.+?) at', str(out)).group(1))
    add_to_db_tasklist(user_id, str(date_time), text, number, event_id)


def remind_in(minutes, call):
    # event_id = event_id_and_message_id[call.message.message_id]
    event_id = get_event_id_and_message_id(call.message.message_id)
    print(event_id)

    # date = parse(get_event_by_id(event_id)[0][2])
    # time = parse(get_event_by_id(event_id)[0][3])
    # print('time: ' + str(time))
    event = get_event_by_id(event_id)[0]
    event_date = parse(' '.join([event[2], event[3]]))

    date_time = event_date - datetime.timedelta(minutes=minutes)
    date_time = date_time.strftime("%H:%M %m%d%y")
    # print(call.message.text)
    add_task(call.from_user.id, call.message.text, date_time, event_id)
    bot.send_message(call.from_user.id, f'Я напомню за {minutes} минут до начала')


def cancel_event(event_id, user_id):
    chatids, numbers = get_chatid_and_task_number(event_id)
    if chatids == None:
        delete_event_db(event_id)
        bot.send_message(user_id, 'Событие отменено:\n')
        return
    for number in numbers:
        cmd = ['atrm', str(number)]
        subprocess.check_output(cmd)
        delete_task(number)

    for chatid in chatids:
        description = get_event_by_id(event_id)
        bot.send_message(int(chatid), 'Событие отменено:\n' + description)
    delete_event_db(event_id)


def get_version():
    (mode, ino, dev, nlink, uid, gid, size, atime, mtime, ctime) = os.stat('main.py')
    msg = "last modified: %s" % time.ctime(mtime)
    return msg


@bot.callback_query_handler(func=lambda call: call.data.startswith(calendar_1.prefix))
def callback_inline(call: CallbackQuery):
    """
    Обработка inline callback запросов
    :param call:
    :return:
    """

    # At this point, we are sure that this calendar is ours. So we cut the line by the separator of our calendar
    name, action, year, month, day = call.data.split(calendar_1.sep)
    # Processing the calendar. Get either the date or None if the buttons are of a different type
    date = telebot_calendar.calendar_query_handler(
        bot=bot, call=call, name=name, action=action, year=year, month=month, day=day
    )
    # There are additional steps. Let's say if the date DAY is selected, you can execute your code. I sent a message.
    if action == "DAY":
        msg = bot.send_message(
            chat_id=call.from_user.id,
            text=f"{date.strftime('%d.%m.%Y')}",
            reply_markup=ReplyKeyboardRemove(),
        )
        process_date_step(msg)

    elif action == "CANCEL":
        cancel_adding_event(call.from_user.id)
        # print(f"{calendar_1}: Cancellation")


@bot.callback_query_handler(func=lambda call: True)  # Реакция на кнопки
def button_callback(call):
    print(call)
    if call.data == ' за 15 минут':
        remind_in(15, call)

    if call.data == ' за час':
        remind_in(60, call)

    if call.data == ' Отменить эвент':
        cancel_event(event_id=get_event_id_and_message_id(call.message.message_id), user_id=call.from_user.id)


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

    try:
        role = get_user_role(message.chat.id)
        if not role:
            add_user(message.chat.id, 'user')

        keyboard_markup = get_keyboard_by_id(message.chat.id)

        bot.send_message(message.chat.id, bot_description, reply_markup=keyboard_markup)

    except:
        add_user(message.chat.id, 'user')
        keyboard_markup = get_keyboard_by_id(message.chat.id)
        bot.send_message(message.chat.id, bot_description, reply_markup=keyboard_markup)


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
    logging.info(message)
    if request == 'События сегодня':
        events = get_events_today_db()
        for i, event in enumerate(events.copy()):
            if datetime.datetime.now().strftime("%H:%M") > parse(event[3]).strftime("%H:%M"):
                del events[i]
        print(events)
        if events:
            process_messages(events, user_id, request)

        else:
            bot.send_message(user_id, "Сегодня ничего не проиходит", reply_markup=markup)

    elif request == 'События завтра':
        events = get_events_by_day_db(tomorrow_date())
        # print(events)
        if events:
            # print(response)
            process_messages(events, user_id, request)
        else:
            bot.send_message(user_id, "Завтра ничего не проиходит", reply_markup=markup)

    elif request == 'События на неделе':
        dt = tomorrow_date()
        weekstart = dt - datetime.timedelta(days=dt.weekday())
        weekend = weekstart + datetime.timedelta(days=6)
        events = get_events_by_period_db(tomorrow_date(), weekend)
        # print(events)
        if events:
            # print(response)
            process_messages(events, user_id, request)
        else:
            bot.send_message(user_id, "На неделе ничего не проиходит", reply_markup=markup)

    elif request == 'Добавить эвент':

        role = get_user_role(user_id)[0][0]
        if role == 'admin' or role == 'client':
            markup = types.ReplyKeyboardMarkup(one_time_keyboard=True, resize_keyboard=True)
            markup.add('Отмена')
            bot.send_message(user_id, "Что за мероприятие?", reply_markup=markup)
            bot.register_next_step_handler(message, add_new_event_proc)
        else:
            bot.send_message(user_id, 'Вы не админ', reply_markup=keyboard)

    elif request == 'Добавить клиента':
        role = get_user_role(user_id)[0][0]
        if role == 'admin':
            markup = types.ReplyKeyboardMarkup(one_time_keyboard=True, resize_keyboard=True)
            markup.add('Отмена')
            bot.send_message(user_id, "Введи айди или перешли сообщение", reply_markup=markup)
            bot.register_next_step_handler(message, add_new_client)

    elif request == 'Версия':
        role = get_user_role(user_id)[0][0]
        if role == 'admin':
            bot.send_message(user_id, get_version())


if __name__ == '__main__':
    init_db()
    init_bot()
    bot.polling()
    # send_message_to_admins()
