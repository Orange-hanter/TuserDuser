import sqlite3
import datetime
import os
from config import admins
import psycopg2


try:
    print(os.getcwd())
    #db_connector = sqlite3.connect("./DB/db.db", check_same_thread=False)
    db_connector = psycopg2.connect(dbname='postgres', user='postgres',
                            password='postgres', host='db', port=5432)
    print("DB connected")
except Exception as e:
    print("DB not connected")
    print(str(e))


def init_db():
    """
    number номер процесса at к которому можно обратиться для отмены
    """
    requests = ["""CREATE TABLE IF NOT EXISTS Events (
                                            id          SERIAL,
                                            description TEXT,
                                            date        date,
                                            time        time,
                                            url         TEXT,
                                            image_id    TEXT
                                            );""",
                """CREATE TABLE IF NOT EXISTS Users (id TEXT UNIQUE, role TEXT);""",
                """CREATE TABLE IF NOT EXISTS TaskList (
                                            id          SERIAL,
                                            user_id     text, 
                                            time        text, 
                                            text        text,
                                            number      INTEGER, 
                                            event_id    text
                                            );""",
                ]
    for admin_id in admins:
        requests.append(f"INSERT INTO users (id, role) VALUES ('{admin_id}', 'admin') ON CONFLICT  DO NOTHING;")

    cursor = db_connector.cursor()
    for request in requests:
        cursor.execute(request)
    db_connector.commit()
    # filling the data storage by random test values
    # put_test_data_to_db()
    # проверка на дублирование
    request = """CREATE UNIQUE INDEX IF NOT EXISTS IDX_EVENT ON events(description, date);"""
    cursor = db_connector.cursor()
    cursor.execute(request)
    cursor.close()


def add_event_db(description, date, time, url, image_id):
    cursor = db_connector.cursor()
    try:
        cursor.execute(
            f"INSERT INTO events (description, date,time , url, image_id) VALUES ('{description}', '{date}','{time}', "
            f"'{url}','{image_id}')")

        db_connector.commit()
        cursor.close()
    except Exception as e:
        print("такое поле уже есть")
        print(str(e))

def delete_event_db(event_id):
    request = f"""DELETE FROM artists_backup
                    WHERE event_id = '{event_id}';"""
    cursor = db_connector.cursor()
    cursor.execute(request)
    cursor.fetchall()
    cursor.close()


def get_event_by_id(event_id):
    request = f"""SELECT * 
                        FROM events 
                        WHERE id = '{event_id}'
                    """
    cursor = db_connector.cursor()
    cursor.execute(request)
    event = cursor.fetchall()
    cursor.close()
    return event


def get_events_by_day_db(date):
    request = f"""SELECT * 
                        FROM events 
                        WHERE date = '{date}'
                    """
    cursor = db_connector.cursor()
    cursor.execute(request)
    event_list = cursor.fetchall()
    cursor.close()
    return event_list


def get_events_today_db():
    return get_events_by_day_db(datetime.date.today())


def get_events_by_period_db(date_start, date_end):
    numdays = date_end - date_start
    events = []
    date_list = [date_start + datetime.timedelta(days=x) for x in range(numdays.days)]
    for date in date_list:
        events.append(get_events_by_day_db(date))

    events = [item for sublist in events for item in sublist]

    return events


def put_test_data_to_db():
    date = datetime.date.today()
    with open("./DB/events.test", 'r') as file:
        for line in file.readlines():
            description = line
            date += datetime.timedelta(days=1)
            cursor = db_connector.cursor()
            cursor.execute(f"INSERT INTO events (description, date) VALUES ('{description}', '{date}')")
        db_connector.commit()
    cursor.close()


def add_to_db_tasklist(chatid, time, text, number, event_id):
    db_connector = sqlite3.connect("./DB/db.db", check_same_thread=False)
    cursor = db_connector.cursor()

    ins = f"""INSERT INTO tasklist (user_id ,time , text, number, event_id)  VALUES ('{chatid}', '{time}', '{text}','{number}', '{event_id}')"""
    cursor.execute(ins)
    db_connector.commit()


def add_user(id, role):
    try:
        db_connector = sqlite3.connect("./DB/db.db", check_same_thread=False)
        cursor = db_connector.cursor()
        ins = f"""INSERT INTO 'users'  VALUES ('{id}', '{role}')"""
        cursor.execute(ins)
        db_connector.commit()
    except Exception as e:
        print("Exception: " + str(e))


def get_user_role(id):
    try:
        request = f"""SELECT role 
                            FROM users 
                            WHERE id = '{id}'
                        """
        cursor = db_connector.cursor()
        cursor.execute(request)
        user_role = cursor.fetchall()
        cursor.close()
        return user_role
    except Exception as e:
        print("Exception: " + str(e))
        return None


def get_chatid_and_task_number(event_id):
    request = f"""SELECT chatid,
                        number  
                        FROM tasklist 
                        WHERE event_id = '{event_id}'
                    """
    cursor = db_connector.cursor()
    cursor.execute(request)

    chatid, number = cursor.fetchall()
    return chatid, number


# it's useless for now
def convert_to_binary_data(filename):
    # Convert digital data to binary format
    with open(filename, 'rb') as file:
        blobData = file.read()
    return blobData


if __name__ == '__main__':
    # put_test_data_to_db()
    init_db()
    add_event_db("Надеюсь получить фидбек :3 ", datetime.date.today(), '20:00', 'https://somegans.site/',
                 'AgACAgQAAxkBAAID-V9pspXlh09fm_U3CCk8-hGGyYw2AALqtTEb9l5IUybZYmLv_8eRGLCPIl0AAwEAAwIAA20AA3SeBwABGwQ')
