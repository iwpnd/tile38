package tests

import (
	"testing"
)

func subTestTestCmd(t *testing.T, mc *mockServer) {
	runStep(t, mc, "WITHIN", testcmd_WITHIN_test)
	runStep(t, mc, "INTERSECTS", testcmd_INTERSECTS_test)
	runStep(t, mc, "INTERSECTS_CLIP", testcmd_INTERSECTS_CLIP_test)
	runStep(t, mc, "ExpressionErrors", testcmd_expressionErrors_test)
	runStep(t, mc, "Expressions", testcmd_expression_test)
}

func testcmd_WITHIN_test(mc *mockServer) error {
	poly := `{
				"type": "Polygon",
				"coordinates": [
					[
						[-122.44126439094543,37.72906137107],
						[-122.43980526924135,37.72906137107],
						[-122.43980526924135,37.73421283683962],
						[-122.44126439094543,37.73421283683962],
						[-122.44126439094543,37.72906137107]
					]
				]
			}`
	poly8 := `{"type":"Polygon","coordinates":[[[-122.4408378,37.7341129],[-122.4408378,37.733],[-122.44,37.733],[-122.44,37.7341129],[-122.4408378,37.7341129]],[[-122.44060993194579,37.73345766902749],[-122.44044363498686,37.73345766902749],[-122.44044363498686,37.73355524732416],[-122.44060993194579,37.73355524732416],[-122.44060993194579,37.73345766902749]],[[-122.44060724973677,37.7336888869566],[-122.4402102828026,37.7336888869566],[-122.4402102828026,37.7339752567853],[-122.44060724973677,37.7339752567853],[-122.44060724973677,37.7336888869566]]]}`
	poly9 := `{"type":"Polygon","coordinates":[[[-122.44037926197052,37.73313523548048],[-122.44017541408539,37.73313523548048],[-122.44017541408539,37.73336857568778],[-122.44037926197052,37.73336857568778],[-122.44037926197052,37.73313523548048]]]}`
	poly10 := `{"type":"Polygon","coordinates":[[[-122.44040071964262,37.73359343010089],[-122.4402666091919,37.73359343010089],[-122.4402666091919,37.73373767596864],[-122.44040071964262,37.73373767596864],[-122.44040071964262,37.73359343010089]]]}`

	return mc.DoBatch([][]interface{}{
		{"SET", "mykey", "point1", "POINT", 37.7335, -122.4412}, {"OK"},
		{"SET", "mykey", "point2", "POINT", 37.7335, -122.44121}, {"OK"},
		{"SET", "mykey", "line3", "OBJECT", `{"type":"LineString","coordinates":[[-122.4408378,37.7341129],[-122.4408378,37.733]]}`}, {"OK"},
		{"SET", "mykey", "poly4", "OBJECT", `{"type":"Polygon","coordinates":[[[-122.4408378,37.7341129],[-122.4408378,37.733],[-122.44,37.733],[-122.44,37.7341129],[-122.4408378,37.7341129]]]}`}, {"OK"},
		{"SET", "mykey", "multipoly5", "OBJECT", `{"type":"MultiPolygon","coordinates":[[[[-122.4408378,37.7341129],[-122.4408378,37.733],[-122.44,37.733],[-122.44,37.7341129],[-122.4408378,37.7341129]]],[[[-122.44091033935547,37.731981251280985],[-122.43994474411011,37.731981251280985],[-122.43994474411011,37.73254976045042],[-122.44091033935547,37.73254976045042],[-122.44091033935547,37.731981251280985]]]]}`}, {"OK"},
		{"SET", "mykey", "point6", "POINT", -5, 5}, {"OK"},
		{"SET", "mykey", "point7", "POINT", 33, 21}, {"OK"},
		{"SET", "mykey", "poly8", "OBJECT", poly8}, {"OK"},

		{"TEST", "GET", "mykey", "point1", "WITHIN", "OBJECT", poly}, {"1"},
		{"TEST", "GET", "mykey", "line3", "WITHIN", "OBJECT", poly}, {"1"},
		{"TEST", "GET", "mykey", "poly4", "WITHIN", "OBJECT", poly}, {"1"},
		{"TEST", "GET", "mykey", "multipoly5", "WITHIN", "OBJECT", poly}, {"1"},
		{"TEST", "GET", "mykey", "poly8", "WITHIN", "OBJECT", poly}, {"1"},

		{"TEST", "GET", "mykey", "point6", "WITHIN", "OBJECT", poly}, {"0"},
		{"TEST", "GET", "mykey", "point7", "WITHIN", "OBJECT", poly}, {"0"},

		{"TEST", "OBJECT", poly9, "WITHIN", "OBJECT", poly8}, {"1"},
		{"TEST", "OBJECT", poly10, "WITHIN", "OBJECT", poly8}, {"0"},
	})
}

func testcmd_INTERSECTS_test(mc *mockServer) error {
	poly := `{
				"type": "Polygon",
				"coordinates": [
					[
						[-122.44126439094543,37.732906137107],
						[-122.43980526924135,37.732906137107],
						[-122.43980526924135,37.73421283683962],
						[-122.44126439094543,37.73421283683962],
						[-122.44126439094543,37.732906137107]
					]
				]
			}`
	poly8 := `{"type":"Polygon","coordinates":[[[-122.4408378,37.7341129],[-122.4408378,37.733],[-122.44,37.733],[-122.44,37.7341129],[-122.4408378,37.7341129]],[[-122.44060993194579,37.73345766902749],[-122.44044363498686,37.73345766902749],[-122.44044363498686,37.73355524732416],[-122.44060993194579,37.73355524732416],[-122.44060993194579,37.73345766902749]],[[-122.44060724973677,37.7336888869566],[-122.4402102828026,37.7336888869566],[-122.4402102828026,37.7339752567853],[-122.44060724973677,37.7339752567853],[-122.44060724973677,37.7336888869566]]]}`
	poly9 := `{"type": "Polygon","coordinates": [[[-122.44037926197052,37.73313523548048],[-122.44017541408539,37.73313523548048],[-122.44017541408539,37.73336857568778],[-122.44037926197052,37.73336857568778],[-122.44037926197052,37.73313523548048]]]}`
	poly10 := `{"type": "Polygon","coordinates": [[[-122.44040071964262,37.73359343010089],[-122.4402666091919,37.73359343010089],[-122.4402666091919,37.73373767596864],[-122.44040071964262,37.73373767596864],[-122.44040071964262,37.73359343010089]]]}`
	poly101 := `{"type":"Polygon","coordinates":[[[-122.44051605463028,37.73375464605226],[-122.44028002023695,37.73375464605226],[-122.44028002023695,37.733903134117966],[-122.44051605463028,37.733903134117966],[-122.44051605463028,37.73375464605226]]]}`

	return mc.DoBatch([][]interface{}{
		{"SET", "mykey", "point1", "POINT", 37.7335, -122.4412}, {"OK"},
		{"SET", "mykey", "point2", "POINT", 37.7335, -122.44121}, {"OK"},
		{"SET", "mykey", "line3", "OBJECT", `{"type":"LineString","coordinates":[[-122.4408378,37.7341129],[-122.4408378,37.733]]}`}, {"OK"},
		{"SET", "mykey", "poly4", "OBJECT", `{"type":"Polygon","coordinates":[[[-122.4408378,37.7341129],[-122.4408378,37.733],[-122.44,37.733],[-122.44,37.7341129],[-122.4408378,37.7341129]]]}`}, {"OK"},
		{"SET", "mykey", "multipoly5", "OBJECT", `{"type":"MultiPolygon","coordinates":[[[[-122.4408378,37.7341129],[-122.4408378,37.733],[-122.44,37.733],[-122.44,37.7341129],[-122.4408378,37.7341129]]],[[[-122.44091033935547,37.731981251280985],[-122.43994474411011,37.731981251280985],[-122.43994474411011,37.73254976045042],[-122.44091033935547,37.73254976045042],[-122.44091033935547,37.731981251280985]]]]}`}, {"OK"},
		{"SET", "mykey", "point6", "POINT", -5, 5}, {"OK"},
		{"SET", "mykey", "point7", "POINT", 33, 21}, {"OK"},
		{"SET", "mykey", "poly8", "OBJECT", poly8}, {"OK"},

		{"TEST", "GET", "mykey", "point1", "INTERSECTS", "OBJECT", poly}, {"1"},
		{"TEST", "GET", "mykey", "point2", "INTERSECTS", "OBJECT", poly}, {"1"},
		{"TEST", "GET", "mykey", "line3", "INTERSECTS", "OBJECT", poly}, {"1"},
		{"TEST", "GET", "mykey", "poly4", "INTERSECTS", "OBJECT", poly}, {"1"},
		{"TEST", "GET", "mykey", "multipoly5", "INTERSECTS", "OBJECT", poly}, {"1"},
		{"TEST", "GET", "mykey", "poly8", "INTERSECTS", "OBJECT", poly}, {"1"},

		{"TEST", "GET", "mykey", "point6", "INTERSECTS", "OBJECT", poly}, {"0"},
		{"TEST", "GET", "mykey", "point7", "INTERSECTS", "OBJECT", poly}, {"0"},

		{"TEST", "OBJECT", poly9, "INTERSECTS", "OBJECT", poly8}, {"1"},
		{"TEST", "OBJECT", poly10, "INTERSECTS", "OBJECT", poly8}, {"1"},
		{"TEST", "OBJECT", poly101, "INTERSECTS", "OBJECT", poly8}, {"0"},
	})
}

func testcmd_INTERSECTS_CLIP_test(mc *mockServer) error {
	poly8 := `{"type":"Polygon","coordinates":[[[-122.4408378,37.7341129],[-122.4408378,37.733],[-122.44,37.733],[-122.44,37.7341129],[-122.4408378,37.7341129]],[[-122.44060993194579,37.73345766902749],[-122.44044363498686,37.73345766902749],[-122.44044363498686,37.73355524732416],[-122.44060993194579,37.73355524732416],[-122.44060993194579,37.73345766902749]],[[-122.44060724973677,37.7336888869566],[-122.4402102828026,37.7336888869566],[-122.4402102828026,37.7339752567853],[-122.44060724973677,37.7339752567853],[-122.44060724973677,37.7336888869566]]]}`
	poly9 := `{"type":"Polygon","coordinates":[[[-122.44037926197052,37.73313523548048],[-122.44017541408539,37.73313523548048],[-122.44017541408539,37.73336857568778],[-122.44037926197052,37.73336857568778],[-122.44037926197052,37.73313523548048]]]}`
	multipoly5 := `{"type":"MultiPolygon","coordinates":[[[[-122.4408378,37.7341129],[-122.4408378,37.733],[-122.44,37.733],[-122.44,37.7341129],[-122.4408378,37.7341129]]],[[[-122.44091033935547,37.731981251280985],[-122.43994474411011,37.731981251280985],[-122.43994474411011,37.73254976045042],[-122.44091033935547,37.73254976045042],[-122.44091033935547,37.731981251280985]]]]}`
	poly101 := `{"type":"Polygon","coordinates":[[[-122.44051605463028,37.73375464605226],[-122.44028002023695,37.73375464605226],[-122.44028002023695,37.733903134117966],[-122.44051605463028,37.733903134117966],[-122.44051605463028,37.73375464605226]]]}`

	return mc.DoBatch([][]interface{}{
		{"SET", "mykey", "point1", "POINT", 37.7335, -122.4412}, {"OK"},

		{"TEST", "OBJECT", poly9, "INTERSECTS", "CLIP", "OBJECT", "{}"}, {"ERR invalid clip type 'OBJECT'"},
		{"TEST", "OBJECT", poly9, "INTERSECTS", "CLIP", "CIRCLE", "1", "2", "3"}, {"ERR invalid clip type 'CIRCLE'"},
		{"TEST", "OBJECT", poly9, "INTERSECTS", "CLIP", "GET", "mykey", "point1"}, {"ERR invalid clip type 'GET'"},
		{"TEST", "OBJECT", poly9, "WITHIN", "CLIP", "BOUNDS", 10, 10, 20, 20}, {"ERR invalid argument 'CLIP'"},

		{"TEST", "OBJECT", poly9, "INTERSECTS", "CLIP", "BOUNDS", 37.732906137107, -122.44126439094543, 37.73421283683962, -122.43980526924135}, {"[1 " + poly9 + "]"},
		{"TEST", "OBJECT", poly8, "INTERSECTS", "CLIP", "BOUNDS", 37.733, -122.4408378, 37.7341129, -122.44}, {"[1 " + poly8 + "]"},
		{"TEST", "OBJECT", multipoly5, "INTERSECTS", "CLIP", "BOUNDS", 37.73227823422744, -122.44120001792908, 37.73319038868677, -122.43955314159392}, {"[1 " + `{"type":"FeatureCollection","features":[{"type":"Feature","geometry":{"type":"Polygon","coordinates":[[[-122.4408378,37.73319038868677],[-122.4408378,37.733],[-122.44,37.733],[-122.44,37.73319038868677],[-122.4408378,37.73319038868677]]]},"properties":{}},{"type":"Feature","geometry":{"type":"Polygon","coordinates":[[[-122.44091033935547,37.73227823422744],[-122.43994474411011,37.73227823422744],[-122.43994474411011,37.73254976045042],[-122.44091033935547,37.73254976045042],[-122.44091033935547,37.73227823422744]]]},"properties":{}}]}` + "]"},
		{"TEST", "OBJECT", poly101, "INTERSECTS", "CLIP", "BOUNDS", 37.73315644825698, -122.44054287672043, 37.73349585185455, -122.44008690118788}, {"0"},
	})
}

func testcmd_expressionErrors_test(mc *mockServer) error {
	return mc.DoBatch([][]interface{}{
		{"SET", "mykey", "foo", "OBJECT", `{"type":"LineString","coordinates":[[-122.4408378,37.7341129],[-122.4408378,37.733]]}`}, {"OK"},
		{"SET", "mykey", "bar", "OBJECT", `{"type":"LineString","coordinates":[[-122.4408378,37.7341129],[-122.4408378,37.733]]}`}, {"OK"},
		{"SET", "mykey", "baz", "OBJECT", `{"type":"LineString","coordinates":[[-122.4408378,37.7341129],[-122.4408378,37.733]]}`}, {"OK"},

		{"TEST", "GET", "mykey", "foo", "INTERSECTS", "(", "GET", "mykey", "bar"}, {
			"ERR wrong number of arguments for 'test' command"},
		{"TEST", "GET", "mykey", "foo", "INTERSECTS", "GET", "mykey", "bar", ")"}, {
			"ERR invalid argument ')'"},

		{"TEST", "GET", "mykey", "foo", "INTERSECTS", "OR", "GET", "mykey", "bar"}, {
			"ERR invalid argument 'or'"},
		{"TEST", "GET", "mykey", "foo", "INTERSECTS", "AND", "GET", "mykey", "bar"}, {
			"ERR invalid argument 'and'"},
		{"TEST", "GET", "mykey", "foo", "INTERSECTS", "GET", "mykey", "bar", "OR", "AND", "GET", "mykey", "baz"}, {
			"ERR invalid argument 'and'"},
		{"TEST", "GET", "mykey", "foo", "INTERSECTS", "GET", "mykey", "bar", "AND", "OR", "GET", "mykey", "baz"}, {
			"ERR invalid argument 'or'"},
		{"TEST", "GET", "mykey", "foo", "INTERSECTS", "GET", "mykey", "bar", "OR", "OR", "GET", "mykey", "baz"}, {
			"ERR invalid argument 'or'"},
		{"TEST", "GET", "mykey", "foo", "INTERSECTS", "GET", "mykey", "bar", "AND", "AND", "GET", "mykey", "baz"}, {
			"ERR invalid argument 'and'"},
		{"TEST", "GET", "mykey", "foo", "INTERSECTS", "GET", "mykey", "bar", "OR"}, {
			"ERR wrong number of arguments for 'test' command"},
		{"TEST", "GET", "mykey", "foo", "INTERSECTS", "GET", "mykey", "bar", "AND"}, {
			"ERR wrong number of arguments for 'test' command"},
		{"TEST", "GET", "mykey", "foo", "INTERSECTS", "GET", "mykey", "bar", "NOT"}, {
			"ERR wrong number of arguments for 'test' command"},
		{"TEST", "GET", "mykey", "foo", "INTERSECTS", "GET", "mykey", "bar", "NOT", "AND", "GET", "mykey", "baz"}, {
			"ERR invalid argument 'and'"},
	})
}

func testcmd_expression_test(mc *mockServer) error {
	poly := `{
				"type": "Polygon",
				"coordinates": [
					[
						[-122.44126439094543,37.732906137107],
						[-122.43980526924135,37.732906137107],
						[-122.43980526924135,37.73421283683962],
						[-122.44126439094543,37.73421283683962],
						[-122.44126439094543,37.732906137107]
					]
				]
			}`
	poly8 := `{"type":"Polygon","coordinates":[[[-122.4408378,37.7341129],[-122.4408378,37.733],[-122.44,37.733],[-122.44,37.7341129],[-122.4408378,37.7341129]],[[-122.44060993194579,37.73345766902749],[-122.44044363498686,37.73345766902749],[-122.44044363498686,37.73355524732416],[-122.44060993194579,37.73355524732416],[-122.44060993194579,37.73345766902749]],[[-122.44060724973677,37.7336888869566],[-122.4402102828026,37.7336888869566],[-122.4402102828026,37.7339752567853],[-122.44060724973677,37.7339752567853],[-122.44060724973677,37.7336888869566]]]}`
	poly9 := `{"type": "Polygon","coordinates": [[[-122.44037926197052,37.73313523548048],[-122.44017541408539,37.73313523548048],[-122.44017541408539,37.73336857568778],[-122.44037926197052,37.73336857568778],[-122.44037926197052,37.73313523548048]]]}`

	return mc.DoBatch([][]interface{}{
		{"SET", "mykey", "line3", "OBJECT", `{"type":"LineString","coordinates":[[-122.4408378,37.7341129],[-122.4408378,37.733]]}`}, {"OK"},
		{"SET", "mykey", "poly8", "OBJECT", poly8}, {"OK"},

		{"TEST", "OBJECT", poly9, "INTERSECTS", "NOT", "OBJECT", poly}, {"0"},
		{"TEST", "OBJECT", poly9, "INTERSECTS", "NOT", "NOT", "OBJECT", poly}, {"1"},
		{"TEST", "OBJECT", poly9, "INTERSECTS", "NOT", "NOT", "NOT", "OBJECT", poly}, {"0"},

		{"TEST", "OBJECT", poly9, "INTERSECTS", "OBJECT", poly8, "OR", "OBJECT", poly}, {"1"},
		{"TEST", "OBJECT", poly9, "INTERSECTS", "OBJECT", poly8, "AND", "OBJECT", poly}, {"1"},
		{"TEST", "OBJECT", poly9, "INTERSECTS", "GET", "mykey", "poly8", "OR", "OBJECT", poly}, {"1"},

		{"TEST", "OBJECT", poly9, "INTERSECTS", "GET", "mykey", "line3"}, {"0"},
		{"TEST", "OBJECT", poly9, "INTERSECTS", "GET", "mykey", "poly8", "AND",
			"(", "OBJECT", poly, "AND", "GET", "mykey", "line3", ")"}, {"0"},
		{"TEST", "OBJECT", poly9, "INTERSECTS", "GET", "mykey", "poly8", "AND",
			"(", "OBJECT", poly, "OR", "GET", "mykey", "line3", ")"}, {"1"},
		{"TEST", "OBJECT", poly9, "INTERSECTS", "GET", "mykey", "poly8", "AND",
			"(", "OBJECT", poly, "AND", "NOT", "GET", "mykey", "line3", ")"}, {"1"},
		{"TEST", "OBJECT", poly9, "INTERSECTS", "NOT", "GET", "mykey", "line3"}, {"1"},
		{"TEST", "NOT", "OBJECT", poly9, "INTERSECTS", "GET", "mykey", "line3"}, {"1"},
		{"TEST", "OBJECT", poly9, "INTERSECTS", "GET", "mykey", "line3",
			"OR", "OBJECT", poly8, "AND", "OBJECT", poly}, {"1"},
		{"TEST", "OBJECT", poly9, "INTERSECTS", "OBJECT", poly8, "AND", "OBJECT", poly,
			"OR", "GET", "mykey", "line3"}, {"1"},
		{"TEST", "OBJECT", poly9, "INTERSECTS", "GET", "mykey", "line3", "OR",
			"(", "OBJECT", poly8, "AND", "OBJECT", poly, ")"}, {"1"},
		{"TEST", "OBJECT", poly9, "INTERSECTS",
			"(", "GET", "mykey", "line3", "OR", "OBJECT", poly8, ")", "AND", "OBJECT", poly}, {"1"},

		{"TEST", "OBJECT", poly9, "WITHIN", "OBJECT", poly8, "OR", "OBJECT", poly}, {"1"},
		{"TEST", "OBJECT", poly9, "WITHIN", "OBJECT", poly8, "AND", "OBJECT", poly}, {"1"},

		{"TEST", "OBJECT", poly9, "WITHIN", "GET", "mykey", "line3"}, {"0"},
		{"TEST", "OBJECT", poly9, "WITHIN", "GET", "mykey", "poly8", "AND",
			"(", "OBJECT", poly, "AND", "GET", "mykey", "line3", ")"}, {"0"},
		{"TEST", "OBJECT", poly9, "WITHIN", "GET", "mykey", "poly8", "AND",
			"(", "OBJECT", poly, "OR", "GET", "mykey", "line3", ")"}, {"1"},
		{"TEST", "OBJECT", poly9, "WITHIN", "GET", "mykey", "poly8", "AND",
			"(", "OBJECT", poly, "AND", "NOT", "GET", "mykey", "line3", ")"}, {"1"},
		{"TEST", "OBJECT", poly9, "WITHIN", "NOT", "GET", "mykey", "line3"}, {"1"},
	})
}
