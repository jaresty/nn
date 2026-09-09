import json, pathlib, os
base=pathlib.Path('/tmp/nn-tracer-replay')
for name in ['project-a','project-b','home','config','cache','notebook']:(base/name).mkdir(parents=True,exist_ok=True)
def msg(id, content, role='assistant', agent=None, **extra):
 r={'type':'message','id':id,'timestamp':'2026-09-01T12:00:00Z','message':{'role':role,'content':content,**extra}}
 if agent:r['agentId']=agent
 return r
a=[{'type':'session','version':3,'id':'fixture-producer'},msg('q','Producer work for fixture-job-1. Keep the wire key task_id.','user')]
for agent in ['A','B','C']:
 a += [msg('launch'+agent,[{'type':'toolCall','id':'call'+agent,'name':'Agent','arguments':{'description':'Implement fixture '+agent,'prompt':'Implement a bounded fixture change; retain evidence.'}}]),{'type':'message','parentId':'launch'+agent,'message':{'role':'toolResult','toolName':'Agent','toolCallId':'call'+agent,'content':'spawned','details':{'status':'background','agentId':agent}}}]
a += [msg('root-result','Producer sample for fixture-job-1 emits {"task_id":"job-1"}; no consumer integration test recorded.'),msg('a-work','Updated producer fixture to emit task_id; checked one serialization example.',agent='A'),msg('b-work','Independent consumer compatibility has not been checked.',agent='B'),msg('c-work','Uninspected branch contains a separate unrelated packaging question.',agent='C')]
b=[{'type':'session','version':3,'id':'fixture-consumer'},msg('qb','Read producer sample fixture-job-1.','user'),msg('callb',[{'type':'toolCall','id':'decode1','name':'bash','arguments':{'command':'python decode_fixture.py'}}]),msg('resultb','Input {"task_id":"job-1"}; consumer expression item["id"] raised KeyError: id.','toolResult',toolName='bash',toolCallId='decode1',isError=True)]
for path, records in [(base/'project-a/producer.jsonl',a),(base/'project-b/consumer.jsonl',b)]:path.write_text(''.join(json.dumps(x)+'\n' for x in records))
print('REPLAY_FIXTURES_READY: /tmp/nn-tracer-replay/project-a/producer.jsonl and /tmp/nn-tracer-replay/project-b/consumer.jsonl')
