# FSH24
FSH24 - Fast Sample Hash 24-byte.<br>
A super fast integrity hash using strategic 4MB sampling.

# The goal of FSH24
I aim to provide a system that allows for rapid checking and testing of large files from the internet.<br>
Game and videos have only ballooned in size over the past few years. Trying to process these files with conventional hashing methods wastes way to much time and resources.<br>
We only really need to test if the file was downloaded correctly, we don't have to nit pick over ever single byte in a file. We just need to know if it's "good enough".<br>
FSH24 aims to allow for robust file integrity checking, without taking 30 mins or more just to see if the game you downloaded is even going to install.

# Pros and Cons
<b>What FSH24 is good at</b>
- Basic integrity test for large files (100MB or more, ideally more then 5GB)
- Fast sampling as each "chunk" is only 4MB. and we only need a handful of samples to verify a file

<b>What FSH24 is NOT good at</b>
- 100% integrity. for eg, bit flipping or other weird edge cases wont be sampled and hashed.
- Malicious attacks. as this method is open source, it would be trivial to edit parts of the file and carve around the sampled sectors.

# How does FSH24 work
