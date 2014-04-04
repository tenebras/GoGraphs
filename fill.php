<?php

$server = 'http://localhost:8080';
$iterations = (int)$argv[1];

$objectId = $graph = $comment = $meta = $value = null;

for( $i=0; $i < $iterations; $i++ )
{
	$graph = 'test_'.mt_rand(0,10);
	$objectId = mt_rand(1, 1000);
	$value = mt_rand(1, 100);
	$comment = mt_rand(0,1)? urlencode("Comment for $graph and object $objectId with value $value"): '';
	
	file_get_contents($server."/push?title=$graph&object_id=$objectId&value=$value&comment=$comment");
	
	if( ( $i >= 1000 && $i % 1000 == 0 ) 
		|| ( $i < 1000 && $i >= 100 && $i % 100 == 0 ) 
		|| ( $i >= 10 && $i < 100 && $i % 10 == 0 )
	)
	{
		echo "Processed: $i\n";
	}
}

echo "Processed: $i [Done]";

// select date_trunc('day', ts), object_id, sum(value), sum(amount) from data where ts between '2014-04-01' and '2014-04-04' and graph_id=4 and object_id=712 group by date_trunc('day', ts), object_id;