from abc import ABC
from bs4 import BeautifulSoup, ResultSet

from iParcer import ParseSite, Item


class BrestTheatre(ParseSite, ABC):

    def __init__(self):
        super().__init__()
        self._URL = "https://bresttheatre.info/afisha.html?limit=20"

    def extract_data(self):
        soup = BeautifulSoup(self.html_text, "lxml")
        cards_in_page = soup.find_all('div', class_="block__poster__info")
        cards = list()
        for record in cards_in_page:
            cards.append(self._extract_card(record))
        return cards

    def _extract_card(self, record) -> Item:
        card = Item()

        block = record.find("div", class_="time__img__text")
        card.date_of_event = block.text

        block = record.find("div", class_="text__only__text")
        card.title = block.find("b").text
        card.title.split()

        card.description = block.find("p").text
        card.description.split()
        return card

    def configure(self, html_text):
        self.html_text = html_text
