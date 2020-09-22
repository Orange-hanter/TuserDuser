import sqlite3
import datetime

conn = sqlite3.connect("./DB/db.db", check_same_thread=False)
print("DB connected")


def init_db():
    request = """CREATE TABLE IF NOT EXISTS events (
                                            id          INTEGER  UNIQUE
                                            PRIMARY KEY AUTOINCREMENT,
                                            description TEXT,
                                            date        DATETIME,
                                            time        DATETIME,
                                            url         TEXT,
                                            image_id       TEXT
                                            );"""
    cursor = conn.cursor()
    cursor.execute(request)
    cursor.execute("""CREATE TABLE IF NOT EXISTS 'users'(id TEXT UNIQUE, role TEXT)""")
    cursor.execute("""CREATE TABLE IF NOT EXISTS 'tasklist'
                                  (id text, time text, text text, uid text, number INTEGER)
                               """)
    conn.commit()
    # filling the data storage by random test values
    # put_test_data_to_db()
    cursor.close()


def add_event_db(description, date, time, url,image_id):
    cursor = conn.cursor()

    cursor.execute(f"INSERT INTO events (description, date,time , url, image_id) VALUES ('{description}', '{date}','{time}', '{url}','{image_id}')")
    conn.commit()
    cursor.close()

def get_event_by_id(event_id):
    request = f"""SELECT * 
                        FROM events 
                        WHERE id = '{event_id}'
                    """
    cursor = conn.cursor()
    cursor.execute(request)
    event = cursor.fetchall()
    cursor.close()
    return event

def get_events_by_day_db(date):
    request = f"""SELECT * 
                        FROM events 
                        WHERE date = '{date}'
                    """
    cursor = conn.cursor()
    cursor.execute(request)
    event_list = cursor.fetchall()
    cursor.close()
    return event_list


def get_events_today_db():
    return get_events_by_day_db(datetime.date.today())


def get_events_by_period_db(date_start, date_end):
    numdays = date_end - date_start
    events =[]
    date_list = [date_start + datetime.timedelta(days=x) for x in range(numdays)]
    for date in date_list:
        events.append(get_events_by_day_db(date))

    return events


def put_test_data_to_db():
    date = datetime.date.today()
    with open("./DB/events.test", 'r') as file:
        for line in file.readlines():
            description = line
            date += datetime.timedelta(days=1)
            cursor = conn.cursor()
            cursor.execute(f"INSERT INTO events (description, date) VALUES ('{description}', '{date}')")
        conn.commit()
    cursor.close()

def add_to_db_tasklist(chatid, time, text, uid, number):  # Функция добавляет данные в таблицу 'tasklist'
    conn = sqlite3.connect("./DB/db.db")
    cursor = conn.cursor()
    ins = f"""INSERT INTO 'tasklist'  VALUES ('{chatid}', '{time}', '{text}', '{uid}','{number}')"""
    cursor.execute(ins)
    conn.commit()


def add_user(id, role):
    try:
        conn = sqlite3.connect("./DB/db.db")
        cursor = conn.cursor()
        ins = f"""INSERT INTO 'users'  VALUES ('{id}', '{role}')"""
        cursor.execute(ins)
        conn.commit()
    except Exception as e:
        print("Exception: " + str(e))

def get_user_role(id):
    try:
        request = f"""SELECT role 
                            FROM users 
                            WHERE id = '{id}'
                        """
        cursor = conn.cursor()
        cursor.execute(request)
        user_role = cursor.fetchall()
        cursor.close()
        return user_role
    except Exception as e:
        print("Exception: " + str(e))
        return None

#it's useless for now
def convert_to_binary_data(filename):
    #Convert digital data to binary format
    with open(filename, 'rb') as file:
        blobData = file.read()
    return blobData


if __name__ == '__main__':
    # put_test_data_to_db()
    init_db()
    add_event_db("Надеюсь получить фидбек :3 ", datetime.date.today(),'20:00', 'https://somegans.site/','AgACAgQAAxkBAAID-V9pspXlh09fm_U3CCk8-hGGyYw2AALqtTEb9l5IUybZYmLv_8eRGLCPIl0AAwEAAwIAA20AA3SeBwABGwQ') # немного крипипаст