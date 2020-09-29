import datetime
import os
import re
import urllib.request

import requests as req
import telebot
from bs4 import BeautifulSoup

from config import token, test_user
from db import add_event_db

bot = telebot.TeleBot(token)


def get_datetime(call):
    match_date = re.search(r'\d{4}-\d{2}-\d{2}', call)
    match_time = re.search(r'\d{2}:\d{2}', call)
    date = datetime.datetime.strptime(match_date.group(), '%Y-%m-%d').date()
    return date, match_time.group()


def parse_kvitki():
    resp = req.get("https://www.kvitki.by/rus/bileti/all/status:insales/city:24814/order:date,asc")

    soup = BeautifulSoup(resp.text, "lxml")
    # raw_html = soup.findAll("div", {"class": "concertslist_page"})[0]
    raw_html = soup.findAll("div", {"class": "event_short"})
    for event in raw_html:
        url_event = \
            re.findall('<a class="event_short_inner event" href=(.*?)><span class="event_short_top"><img alt=""',
                       str(event))[0]
        name_event = re.findall('<span class="event_short_title"> (.*?)</span></span></span>', str(event))[0]
        match = re.findall('datetime=.*"><span class="mobile_layout_list_mobile_hidden">', str(event))[0]
        img_url = re.findall(' data-lazysrc=(.*?)src', str(event))[0]

        date, time = get_datetime(match)
        print(url_event)
        img_path = f'DB/images/{name_event}.jpg'
        urllib.request.urlretrieve(img_url[1:-2], img_path)
        print(img_path)

        api_ret = bot.send_photo(test_user, open(img_path, 'rb'))
        photo_id = api_ret.photo[0].file_id
        os.remove(img_path)
        # срезы url это временное решение
        add_event_db(name_event, date, time, url_event[1:-1], photo_id)


if __name__ == '__main__':
    parse_kvitki()
