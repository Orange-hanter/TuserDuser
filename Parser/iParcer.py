from abc import ABC, abstractmethod


class Item:
    picture = None
    title = "No title"
    description = "No Description"
    date_of_event = '17.03.1994 00:00'
    link = ""

    def __repr__(self):
        return self.title + '\n' + self.date_of_event + '\n' + self.description + '\n'


class ParseSite(ABC):

    def __init__(self):
        self._URL = ''
        self.html_text = None

    """ Return list of Items that contains all information about event
    """
    @abstractmethod
    def extract_data(self) -> list:
        pass

    @abstractmethod
    def configure(self, html_text):
        pass

    @property
    def url(self) -> str:
        return self._URL


