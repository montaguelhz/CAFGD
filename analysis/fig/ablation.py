import matplotlib
import pandas as pd
import seaborn as sns
import matplotlib.pyplot as plt

fgd = [153194782,153839210,153640039,153640039,152885618,153640039,153659118,152877377,155340493,153640039]
cafgd1 = [154158863,154836943,155822383,154083188,154600452,154768713,153747042,153875548,154300953,154149419]
cafgd2 = [154711666,155445516,154389001,155050233,154646721,156541931,154348660,154179850,154613262,155095306]
cafgd  = [155976464,155884141,155928062,154672718,156391987,155047712,156345071,156014247,156525107,154234942]
random = [139231066,138910672,140556005,138663749,139346783,139258953,138707007,139446010,141389769,138927602]
scores = [fgd,cafgd1,cafgd2,cafgd]
optimal = 168468657
x = ['FGD','CAFGD-M','CAFGD-M&B','CAFGD']
value = []
label = []
data = {
    'value': value,
    'label': label
}
rs = 0
for s in random:
    rs+=s
rs/=len(random)
pre = 100
for i in range(0,len(scores)):
    score = scores[i]
    count=0
    for s in score:
        value.append((optimal-s)/optimal*100)
        count+=(optimal-s)/optimal*100
        label.append(x[i])
    print(x[i])
    print((pre - count/10)/pre)
    pre = count/10



plt.figure(figsize=(6, 3), dpi=120)
data = pd.DataFrame(data)
bars = sns.barplot(data=data,x='label',y='value',hue_order=x, errorbar='sd', edgecolor="0")
hatches = [ "/" , "\\" , "|" , "-" , "+" , "x", "o", "O", ".", "*" ]
for i, bar in enumerate(bars.patches):
    hatch_m = hatches[i]
    bar.set_hatch(hatch_m)
plt.xlabel('')
plt.ylabel('Unutilized GPU (%)')
plt.ylim(5,10)

plt.savefig("unutilized_ablation", bbox_inches='tight')