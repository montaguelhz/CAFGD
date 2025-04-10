# 
# Usage: python3 generate_run_scripts.py > run_scripts.sh


DATE = "2025_04_10" # Used as the folder name under experiments/ to hold all log results. To avoid collision of repeated experiments, may change date or append _v1, _v2, etc.
REMARK = "Artifacts"
REPEAT = 1 # Number of repetitive experiments.
FILELIST = [
    # NOTE(lhz): the config of CAFGD need to be modified in the code.

    # dynamic
    "gpudata/spec_task",

    # static
    "data/openb_pod_list_default",

    # specific workloads
    "data/openb_pod_list_gpuspec10",
    "data/openb_pod_list_gpuspec20",
    "data/openb_pod_list_gpuspec25",
    "data/openb_pod_list_gpuspec33",

    "data/openb_pod_list_multigpu20",
    "data/openb_pod_list_multigpu30",
    "data/openb_pod_list_multigpu40",
    "data/openb_pod_list_multigpu50",

    # NOTE(lhz): the other workloads are too small to use. 
]

AllMethodList = [
    ["01", "Random", "random", "<none>", "<none>"],
    ["02", "DotProd", "best", "merge", "max"],
    ["03", "GpuClustering", "<none>", "<none>", "<none>"],
    ["04", "GpuPacking", "<none>", "<none>", "<none>"],
    ["05", "BestFit", "<none>", "<none>", "<none>"],
    ["06", "FGD", "<self>", "share", "max"],
    ["07", "CAFGD", "<self>", "share", "max"],
]

AllMethodDict = {}
for item in AllMethodList:
    AllMethodDict[item[0]] = item

#####################################################################
#####################################################################
#####################################################################

MethodList = AllMethodList.copy()

MethodList = [
    ["01", "Random", "random", "<none>", "<none>"],
    ["02", "DotProd", "best", "merge", "max"],
    ["03", "GpuClustering", "<none>", "<none>", "<none>"],
    ["04", "GpuPacking", "<none>", "<none>", "<none>"],
    ["05", "BestFit", "<none>", "<none>", "<none>"],
    ["06", "FGD", "<self>", "share", "max"],
    ["07", "CAFGD","<self>", "share", "max"],
]

def get_dir_name_from_method(method_input):
    if len(method_input) != 5:
        print("[ERROR] get_dir_name_from_method: len(method) == 5, including id, policy, gsm, dem, nm")
        return "default_name"
    id, policy, gsm, dem, nm = method_input
    gsm = policy if gsm == "<self>" else gsm # no need to adjust, except that <self> is not allowed in bash. generate_config_and_run will recover the policy's full name
    dir_name = "%s-%s" % (id, policy)
    suffix = ""
    suffix += '_%s' % gsm if gsm != "<none>" else ''
    suffix += '_%s' % dem if dem != "<none>" else ''
    suffix += '_%s' % nm if nm != "<none>" else ''
    return dir_name # + suffix

def get_method_from_policy_id_list(id_list):
    if type(id_list) == list:
        return [AllMethodDict.get("%02d" % id, None) if type(id)==int else AllMethodDict.get("%s" % id, None) for id in id_list]
    else:
        return [AllMethodDict.get("%02d" % id_list, None) if type(id)==int else AllMethodDict.get("%s" % id_list, None)]

def get_dir_name_from_policy_id_list(id_list):
    return [get_dir_name_from_method(x) for x in get_method_from_policy_id_list(id_list)]

###########################################################
###########################################################
###########################################################

def generate_run_scripts(asyncc=True, parallel=10):
    DateAndRemark = DATE + "-" + REMARK.replace(' ', "_").replace('(',"_").replace(')',"_")
    numJobs=0
    if asyncc:
        print('#!/bin/bash\n# screen -dmS sim-%s bash -c "bash run_scripts_%s.sh"\n' % (DateAndRemark, DATE[-4:]))
    else:
        print('#!/bin/bash\n# cat run_scripts_%s.sh | while read i; do printf "%%q\\n" "$i"; done | xargs --max-procs=16 -I CMD bash -c CMD\n' % (DATE[-4:]))

    for time in range(0,10):
        for file in FILELIST:
            filename = file.split('/')[-1]
            for id, policy, gsm, dem, nm in MethodList:  # GpuSelMethod, DimExtMethod, NormMethod
                dir_name = get_dir_name_from_method([id, policy, gsm, dem, nm])
                gsm = policy if gsm == "<self>" else gsm
                OUTPUT_YAML = False
                SHUFFLE_POD = False
                outstr = "# %s, %s, %s, %s, %s @ %s\n" % (id, policy, gsm, dem, nm, filename)
                outstr += 'EXPDIR="experiments/%s/%s/%s/%s' % (DATE, filename, dir_name, time)
                outstr += '" && mkdir -p ${EXPDIR} && touch "${EXPDIR}/terminal.out" && '
                outstr += 'python3 scripts/generate_config_and_run.py -d "${EXPDIR}" '
                outstr += '-e -b '
                outstr += '-f %s ' % file
                outstr += '-%s 1000 ' % policy
                outstr += '-gpusel %s ' % gsm if gsm != "<none>" else ''
                outstr += '-dimext %s ' % dem if dem != "<none>" else ''
                outstr += '-norm %s ' % nm if nm != "<none>" else ''
                # NOTE(lhz): we keep the time sequence
                outstr += '-tune 0 ' 
                outstr += '-tuneseed 42 '
                outstr += "--shuffle-pod=true " if SHUFFLE_POD else ""
                outstr += '-y "${EXPDIR}/snapshot/yaml" ' if OUTPUT_YAML else ""
                outstr += '-z "${EXPDIR}/snapshot/ds01" '
                outstr += '| tee -a "${EXPDIR}/terminal.out" '
                outstr += '&& python3 scripts/analysis.py -f -g ${EXPDIR} '
                outstr += '| tee -a "${EXPDIR}/terminal.out" '
                if asyncc:
                    outstr += " &"
                print(outstr + "\n")

                numJobs += 1
                if asyncc and (numJobs % parallel == 0):
                    print("date & wait\n")  # force them to sync
    if asyncc:
        print("wait && date")

if __name__=='__main__':
    # generate_run_scripts(asyncc=True)
    #: $ bash run_scripts.txt
    generate_run_scripts(asyncc=True)
    #: $ cat run_scripts.txt | while read i; do printf "%q\n" "$i"; done | xargs --max-procs=16 -I CMD bash -c CMD

