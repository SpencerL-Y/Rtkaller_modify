import os
import sys
import numpy as np
import pandas as pd
from pandas import Series, DataFrame
import matplotlib.pyplot as plt

exeCoverLock1 = []
exeCoverLock2 = []
exeCoverLock3 = []
exeCoverLock4 = []
exeCoverOrig1 = []
exeCoverOrig2 = []
exeCoverOrig3 = []
def recordFile(filename, resultList):
    with open(filename, encoding='utf-8') as file:
        for line in file:
            if line.find("running") == -1 and line.find("failed") == -1:
                lineList = line.split(",")
                if(len(lineList) > 0):
                    exe = int(lineList[1].strip().split(" ")[1].strip())
                    cover = int(lineList[2].strip().split(" ")[1].strip())
                    resultList.append((exe, cover))
                    # print((exe, cover))

if __name__ == "__main__":
    # printFile()

    recordFile("lock_1.csv", exeCoverLock1)
    recordFile("lock_2.csv", exeCoverLock2)
    recordFile("lock_3.csv", exeCoverLock3)
    recordFile("lock_4.csv", exeCoverLock4)
    recordFile("lock_1_ori.csv", exeCoverOrig1)
    recordFile("lock_2_ori.csv", exeCoverOrig2)
    recordFile("lock_3_ori.csv", exeCoverOrig3)

    LockX1 = []
    LockY1 = []
    for pair in exeCoverLock1:
        LockX1.append(pair[0])
        LockY1.append(pair[1])
    LockX2 = []
    LockY2 = []
    for pair in exeCoverLock2:
        LockX2.append(pair[0])
        LockY2.append(pair[1])
    LockX3 = []
    LockY3 = []
    for pair in exeCoverLock3:
        LockX3.append(pair[0])
        LockY3.append(pair[1])
    LockX4 = []
    LockY4 = []
    for pair in exeCoverLock4:
        LockX4.append(pair[0])
        LockY4.append(pair[1])
    OrigX1 = []
    OrigY1 = []
    for pair in exeCoverOrig1:
        OrigX1.append(pair[0])
        OrigY1.append(pair[1])
    OrigX2 = []
    OrigY2 = []
    for pair in exeCoverOrig2:
        OrigX2.append(pair[0])
        OrigY2.append(pair[1])
    OrigX3 = []
    OrigY3 = []
    for pair in exeCoverOrig3:
        OrigX3.append(pair[0])
        OrigY3.append(pair[1])
    plt.plot(LockX1, LockY1, color='r')
    plt.plot(LockX2, LockY2, color='r')
    plt.plot(LockX3, LockY3, color='r')
    plt.plot(LockX4, LockY4, color='r')
    plt.plot(OrigX1, OrigY1, color='b')
    plt.plot(OrigX2, OrigY2, color='b')
    plt.plot(OrigX3, OrigY3, color='b')
    plt.show()