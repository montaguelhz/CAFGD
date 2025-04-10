import matplotlib
import pandas as pd
import seaborn as sns
import matplotlib.pyplot as plt

path=""

total = 213197618

cafgd = [155976464,155884141,155928062,154672718,156391987,155047712,156345071,156014247,156525107,154234942]

fgd = [153194782,153839210,153640039,153640039,152885618,153640039,153659118,152877377,155340493,153640039]

best =[151415435,153170990,152406958,152784065,152069815,151157806,151788843,151640130,151651750,152005534]
best_gpu=[153862706,154091577,153892547,153933802,155146381,155150529,154197874,155159971,153974905,154499680]

dot = [153299380,153316563,154720815,152939104,152705609,153282928,153711982,152577760,153956085,152624208]

pack = [149997692,149997692,147151765,148819053,150371696,148897103,149997692,149997692,149997692,149997692]

cluster = [154380885,152365647,153034868,154380885,154380885,152322352,153541530,153134899,154380885,154380885]
random = [139231066,138910672,140556005,138663749,139346783,139258953,138707007,139446010,141389769,138927602]
scores = [random,pack,best,dot,cluster,fgd,cafgd]
valid_x = ['Random','Packing','Bestfit','DotProd','Clustering','FGD','CAFGD']
value = []
label = []
counts = []
data = {
    'value': value,
    'label': label
}

for i in range(0,len(scores)):
    score = scores[i]
    count=0
    for s in score:
        value.append(s/3600)
        count+=s
        label.append(valid_x[i])

    counts.append(count/36000)
    print(count/36000)


plt.figure(figsize=(6, 6), dpi=120)
data = pd.DataFrame(data)
bars = sns.barplot(data=data,x='label',y='value',hue_order=valid_x, errorbar='sd', edgecolor="0")
hatches = [ "/" , "\\" , "|" , "-" , "+" , "x", "o", "O", ".", "*" ]
for i, bar in enumerate(bars.patches):
    hatch_m = hatches[i]
    bar.set_hatch(hatch_m)
plt.xlabel('')
plt.ylabel('GPU Usage ($GPU \cdot h$)')
plt.ylim(38000,44000)
plt.savefig("gpu_usage", bbox_inches='tight')