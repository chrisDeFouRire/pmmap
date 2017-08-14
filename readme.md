# Poor Man's distributed Map-Reduce, but without reduce

**Why you need PMmap**

Because it's easier to implement a webhook to perform a single task, and delegate batching, concurrency management and asynchronous completion monitoring to an external system.

PMmap is a micro service. You often have to manage "large" batches, like sending 300 emails or crawling 7000 servers.
Now you can delegate this job to PMmap. Send it the work to do and the desired concurrency, and it will handle completion monitoring and results gathering for you.

You send inputs to it (like user ids or server URLs) and for each of these, it calls a webhook on your system to perform the task, gathers results and waits for completion of the whole batch, plus you can tell PMmap to run at most `n` calls to the webhook concurrently.

**It's not a job queue**

Job queues solve only half of the problem:

- you don't get results back
- you can't monitor job completion

Queues are more suited for continuous processes ("create thumbnail images") than batches ("process this batch of data and give me all the results when it's done").

PMmap is more like a map-reduce job, but without the reduce job. Your webhook implements the map function.

**Why Poor Man's?**

To satisfy KISS (keep it simple, stupid), PMmap does the following:

- it stores all inputs in memory until they're sent to your backend (with a bounded limit)
- it is not persistent
- it does persist outputs to disk however, to reduce memory footprint
- it is a single point of failure (ie. you can't have a cluster of PMmap servers)
- it is single-tenant
- it is supposedly deployed with docker to provide security-isolation (ie. *don't expose its port to the internet*)

# The API

The PMmap server is not secured, there's no authentication: anyone can call any route. So don't expose PMmap's port to the internet.

The server listens to `localhost:8080` only. 

[Use docker to deploy it](https://hub.docker.com/r/tlsproxy/pmmap/).

```
docker pull tlsproxy/pmmap
docker run -d --restart=always --name pmmap tlsproxy/pmmap
```

Then link the `pmmap` container and use `http://pmmap:8080` for your HTTP requests, or else expose its 8080 port (`-p 8080:8080`).

## `POST /job` Creates a job 

To create a job, you simply send a JSON to this endpoint.

```
{
	"secret": "a secret string",
	"url": "the url of your webhook",
	"concurrency": 5,
	"maxsize": 1000 
}
```

- the `secret` string will be sent back to your webhook (in the `PMMAP-auth` header) to provide a minimum of security.

- the `url` is the url of your backend. Each input will be `POST`ed to `url/{key}`. Inputs are a key-value pair. The value is sent to the backend in the request-body.

- `concurrency` is the maximum number of inflight requests to your backend.

- `maxsize` is the max number of inputs stored in memory by PMmap. If you send more inputs, PMmap will block until the backend has processed some inputs (processing starts immediately after you send the first input).

The server should reply with a status code of `201 CREATED`. The reply body is a JSON with the same structure as the next route.

## `GET /job/{id}` Gets the job details 

Call this endpoint to get details about the job.
```
{
	"id": "the id of your job",
	"inputs": <int> the number of inputs received,
	"outputs": <int> the number of outputs received,
	"url": "the url of your webhook"
}
```

`inputs` and `outputs` can be used to count outputs already received (ie. replies from your servers).

The server should reply with `200 OK`.

## `PUT /job/{id}/input` Adds inputs to the job 

Once a job is created, you must send input documents to it. To do so, you must post a JSON array of documents

```
[{
	key: "a key", 
	value: <any JSON primitive>
}]
```

The `key` must be unique. You can call this route more than once to add inputs.

As soon as some inputs are sent to PMmap, processing by your backend starts asynchronously and results are stored by PMmap.

If the `maxsize` of the job is too small, this call will block until PMmap has received enough replies from your backend.

The server should reply with `201 CREATED` and return the job in the JSON reply body. See above for structure.

## `POST /job/{id}/complete` Tells the job it has received all inputs 

Because jobs are finite in size, you must tell PMmap when all inputs have been sent and no more will arrive. It's a big difference vs. a work queue.

PMmap should reply with `200 OK` and return the job as a JSON reply.

## `GET /job/{id}/output` Gets the jobs output 

After POSTing to the `complete` route above, you can get the results from the job. 
This route will wait until the job is complete, until all outputs are received from the backend, before sending outputs.

The outputs are sent as a JSON array:

```
[{
	"key": "the key as sent to the backend",
	"value": <any JSON output sent by the backend>
}]
```

PMmap should reply with `200 OK`.

## `DELETE /job/{id}` Deletes the job 

After the job is complete and outputs are read, you should delete the job with this route.