# copydis

redis-benchmark -t set,get  -p 6399
====== SET ======
  100000 requests completed in 1.05 seconds
  50 parallel clients
  3 bytes payload
  keep alive: 1

99.08% <= 1 milliseconds
99.93% <= 2 milliseconds
100.00% <= 3 milliseconds
95238.10 requests per second

====== GET ======
  100000 requests completed in 0.92 seconds
  50 parallel clients
  3 bytes payload
  keep alive: 1

99.82% <= 1 milliseconds
100.00% <= 2 milliseconds
109170.30 requests per second
