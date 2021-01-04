#! /usr/bin/env python2

import datetime
import os
import requests
import sys
import traceback

from requests.exceptions import HTTPError

PT_PROJID_PRI = os.getenv('PT_PROJID_PRIMARY')
PT_PROJID_STR = os.getenv('PT_PROJID_STORAGE')
PT_URLPFX_FMT = 'https://www.pivotaltracker.com/services/v5/projects/{}'
PT_URLPFX_PRI = PT_URLPFX_FMT.format(PT_PROJID_PRI)
PT_HDRS_BASE = {'X-TrackerToken':os.getenv('PT_API_TOKEN')}
PT_PRMS_BASE = {'date_format':'millis', 'limit':333}
PT_DAYS_AGED = 190 if 'PT_DAYS_AGED' not in os.environ else int(os.getenv('PT_DAYS_AGED'))

if PT_DAYS_AGED < 30:
    PT_DAYS_AGED = 30

def mergeDicts(base, added):
    ps = base.copy()
    ps.update(added)
    return ps

def updBefFilter(daysAged):
    date = datetime.datetime.now() - datetime.timedelta(days=daysAged)
    return 'updated_before:{}'.format(date.strftime('%m/%d/%Y'))

# gather stories by limit set in PT_PRMS_BASE or max allowed by API, whichever is lower
def getAgedStories(): 
    filterVal = ', '.join(['state:unscheduled', updBefFilter(PT_DAYS_AGED)])
    params = mergeDicts(PT_PRMS_BASE, {'filter':filterVal})
    response = requests.get(PT_URLPFX_PRI+'/stories', headers=PT_HDRS_BASE, params=params)
    response.raise_for_status()
    return response.json()

def moveStory(story):
    storyID = str(story['id'])
    payload = {'project_id':int(PT_PROJID_STR)}
    response = requests.put(PT_URLPFX_PRI+'/stories/'+storyID, headers=PT_HDRS_BASE, json=payload)
    response.raise_for_status()

def main():
    try:
        agedStories = getAgedStories()
        for story in agedStories:
            moveStory(story)
    except HTTPError, e:
        print(e)

if __name__ == '__main__':
    main()
