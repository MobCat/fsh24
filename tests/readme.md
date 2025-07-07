# Unit tests
I'm not a huge fan of unit tests and would prefer to just eat my own cat food for the best real world test coverage<br>
But I couldn't really think of a good way to on purpose brake hundreds of GBs of downloads to then hash and test them individually.<br><br>
UnitTest4.log was kind of just hay mr AI, what do you think of this code, and it when all word salad fuzzer mode on me<br>
So not really useful outside of telling me I had a bug on how many sample sectors we should be doing, but the bug
causes us to over sample, which imo is preferred to get a little more then 1% coverage then less.<br>
The "bug" is repeatable aka the "bad" math is always the same so it doesn't affect the hash in anyway negative.
The hash is always the same with a little more then 1% coverage of the file.
