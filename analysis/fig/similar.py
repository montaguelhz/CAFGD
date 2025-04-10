import matplotlib
import pandas as pd
import seaborn as sns
import matplotlib.pyplot as plt

fgd50k = [0.151354,0.331957,0.523867,0.651324,0.762555,0.842300,0.5478658678088468]
fgd500k = [0.159380,0.308577,0.487775,0.606073,0.726593,0.800325,0.514986089428015]
cafgd50k = [0.303278,0.505111,0.601445,0.671505,0.765730,0.846509,0.6215867993133779]
cafgd500k = [0.269898,0.504321,0.575684,0.644714,0.739253,0.818137,0.6022163908417176]
src = [fgd50k,fgd500k,cafgd50k,cafgd500k]
src = [fgd500k,cafgd500k]
name = ['fgd50k','fgd500k','cafgd50k','cafgd500k']
name = ['target workload','predict workload']
label = ["$[0,10^1)$","$[10^1,10^2)$","$[10^2,10^3)$","$[10^3,10^4)$","$[10^4,10^5)$","$[10^5,\infty)$","ALL"]
data = {
    'value': [],
    'label': [],
    'src':[]
}
for i in range(0,len(src)):
    s = src[i]
    n = name[i]
    for j in range(0,len(label)):
        data['value'].append(s[j]*100)
        data['label'].append(label[j])
        data['src'].append(n)
data = pd.DataFrame(data)
bars = sns.barplot(data=data,x='label',y='value',hue='src',hue_order=name, order=label, edgecolor="0")

hatches = [  "x","/" ]
num_policy = len(name)
num_groups = len(bars.patches) // num_policy
for i, bar in enumerate(bars.patches):
    hatch_i = (i) // num_groups
    hatch_m = hatches[hatch_i % len(hatches)]
    bar.set_hatch(hatch_m)
bars.bar_label(bars.containers[1], label_type='edge', fmt='%0.1f%%', padding=5)
plt.xlabel('Workloads in different run-time ranges (s)')
plt.ylabel('Similarity (%)')
plt.legend(ncol=1, loc='upper left')
plt.ylim(0,100)
plt.savefig("similarity-500k", bbox_inches='tight')
