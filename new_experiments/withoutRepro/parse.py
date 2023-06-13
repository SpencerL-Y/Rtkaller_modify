import os
import sys
import numpy as np
import pandas as pd
from pandas import Series, DataFrame
import matplotlib.pyplot as plt




syzLock1 = []
syzOrig1 = []
syzOrig2 = []
rtLock1 = []
rtLock2 = []
rtOrig1 = []
rtOrig2 = []

combined1 = []
combined2 = []
combined3 = []
combined4 = []
combined5 = []
def syzRecordFile(filename, resultList):
    with open(filename, encoding='utf-8') as file:
        for line in file:
            if line.find("running") == -1 and line.find("failed") == -1:
                lineList = line.split(",")
                if(len(lineList) > 0):
                    exe = int(lineList[1].strip().split(" ")[1].strip())
                    cover = int(lineList[2].strip().split(" ")[1].strip())
                    resultList.append((exe, cover))
                    print((exe, cover))

def rtRecordFile(filename, resultList):
    with open(filename, encoding='utf-8') as file:
        for line in file:
            if line.find("running") == -1 and line.find("failed") == -1 and line.find("reproducing") == -1 and line.find("machine") == -1 and line.find("runInstance") == -1 and line.find("panic") == -1:
                lineList = line.split(",")
                if(len(lineList) > 0):
                    exe = int(lineList[1].strip().split(" ")[-1].strip())
                    cover = int(lineList[2].strip().split(" ")[-1].strip())
                    resultList.append((exe, cover))
                    print((exe, cover))

if __name__ == "__main__":
    # printFile()

    syzRecordFile("lock_1_withoutRep.csv", syzLock1)
    syzRecordFile("orig_1_withoutRep.csv", syzOrig1)
    syzRecordFile("orig_2_withoutRep.csv", syzOrig2)
    rtRecordFile("Rtkaller_lock_1_with_rep.csv", rtLock1)
    rtRecordFile("Rtkaller_lock_2_with_rep.csv", rtLock2)
    rtRecordFile("Rtkaller_orig_1_with_rep.csv", rtOrig1)
    rtRecordFile("Rtkaller_orig_2_with_rep.csv", rtOrig2)
    rtRecordFile("1.csv", combined1)
    rtRecordFile("2.csv", combined2)
    rtRecordFile("3.csv", combined3)
    rtRecordFile("4.csv", combined4)
    rtRecordFile("5.csv", combined5)

    LockX1 = []
    LockY1 = []
    for pair in syzLock1:
        LockX1.append(pair[0])
        LockY1.append(pair[1])
    LockX2 = []
    LockY2 = []
    for pair in syzOrig1:
        LockX2.append(pair[0])
        LockY2.append(pair[1])
    LockX3 = []
    LockY3 = []
    for pair in syzOrig2:
        LockX3.append(pair[0])
        LockY3.append(pair[1])
    RtLockX1 = []
    RtLockY1 = []
    for pair in rtLock1:
        RtLockX1.append(pair[0])
        RtLockY1.append(pair[1])
    RtLockX2 = []
    RtLockY2 = []
    for pair in rtLock2:
        RtLockX2.append(pair[0])
        RtLockY2.append(pair[1])
    RtOrigX1 = []
    RtOrigY1 = []
    for pair in rtOrig1:
        RtOrigX1.append(pair[0])
        RtOrigY1.append(pair[1])
    RtOrigX2 = []
    RtOrigY2 = []
    for pair in rtOrig2:
        RtOrigX2.append(pair[0])
        RtOrigY2.append(pair[1])
    
    combinedX1 = []
    combinedY1 = []
    for pair in combined1:
        combinedX1.append(pair[0])
        combinedY1.append(pair[1])


    combinedX2 = []
    combinedY2 = []
    for pair in combined2:
        combinedX2.append(pair[0])
        combinedY2.append(pair[1])

    
    combinedX3 = []
    combinedY3 = []
    for pair in combined3:
        combinedX3.append(pair[0])
        combinedY3.append(pair[1])


    combinedX4 = []
    combinedY4 = []
    for pair in combined4:
        combinedX4.append(pair[0])
        combinedY4.append(pair[1])

    
    combinedX5 = []
    combinedY5 = []
    for pair in combined5:
        combinedX5.append(pair[0])
        combinedY5.append(pair[1])


    plt.plot(LockX1, LockY1, color='r')
    plt.plot(LockX2, LockY2, color='r')
    plt.plot(LockX3, LockY3, color='r')
    plt.plot(RtLockX1, RtLockY1, color='y')
    plt.plot(RtLockX2, RtLockY2, color='y')
    plt.plot(RtOrigX1, RtOrigY1, color='g')
    plt.plot(RtOrigX2, RtOrigY2, color='g')
    plt.plot(combinedX1, combinedY1, color="b")
    plt.plot(combinedX2, combinedY2, color="b")
    plt.plot(combinedX3, combinedY3, color="b")
    plt.plot(combinedX4, combinedY4, color="b")
    plt.plot(combinedX5, combinedY5, color="b")
    plt.show()