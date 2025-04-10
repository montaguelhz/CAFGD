import matplotlib
import pandas as pd
import seaborn as sns
import matplotlib.pyplot as plt
cafgd = [155976464,155884141,155928062,154672718,156391987,155047712,156345071,156014247,156525107,154234942]

fgd = [153194782,153839210,153640039,153640039,152885618,153640039,153659118,152877377,155340493,153640039]

best =[151415435,153170990,152406958,152784065,152069815,151157806,151788843,151640130,151651750,152005534]

dot = [153299380,153316563,154720815,152939104,152705609,153282928,153711982,152577760,153956085,152624208]

pack = [149997692,149997692,147151765,148819053,150371696,148897103,149997692,149997692,149997692,149997692]

cluster = [154380885,152365647,153034868,154380885,154380885,152322352,153541530,153134899,154380885,154380885]
random = [139231066,138910672,140556005,138663749,139346783,139258953,138707007,139446010,141389769,138927602]
scores = [random,pack,best,dot,cluster,fgd,cafgd][::-1]
labels = ['Random','Packing','Bestfit','DotProd','Clustering','FGD','CAFGD'][::-1]
value = []
label = []
counts = []
data = {
    'value': value,
    'label': label
}
optimal = 168468657
for i in range(0,len(scores)):
    score = scores[i]
    count=0
    for s in score:
        value.append((optimal-s)/optimal*100)
        count+=s
        label.append(labels[i])
    print(labels[i])
    print(count/10)
    counts.append(count/10)
for c in counts:
    print(1-(optimal-counts[0])/(optimal-c))
plt.figure(figsize=(6, 6), dpi=120)
data = pd.DataFrame(data)
bars = sns.barplot(data=data,x='label',y='value',hue_order=label[::-1], errorbar='sd', edgecolor="0")
hatches = [ "/" , "\\" , "|" , "-" , "+" , "x", "o", "O", ".", "*" ]
for i, bar in enumerate(bars.patches):
    hatch_m = hatches[i]
    bar.set_hatch(hatch_m)
plt.xlabel('')
plt.ylabel('Unutilized GPU (%)')
plt.ylim(6,18)

plt.savefig("unutilized_dynamic", bbox_inches='tight')