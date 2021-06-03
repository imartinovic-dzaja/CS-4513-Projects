CS4513: Project 1 Goat File System
==================================

Note, this document includes a number of design questions that can help your implementation. We highly recommend that you answer each design question **before** attempting the corresponding implementation.
These questions will help you design and plan your implementation and guide you towards the resources you need.
Finally, if you are unsure how to start the project, we recommend you visit office hours for some guidance on these questions before attempting to implement this project.


Team members
-----------------

1. Ivan Martinovic (imartinovic@wpi.edu)

Design Questions
------------------

1. When implementing the `debug()` function, you will need to load the file system via the emulated disk and retrieve the information for superblock and inodes.
1.1 How will you read the superblock?
	-Create a Block block variable to read the superblock into
	-Call wread(0, block)

1.2 How will you traverse all the inodes?
	-You need to check how many inode blocks there exist on the disk, by reading the data from the superblock
	-Then read each inode block into the block variable
		-for each of those inode blocks create an Inode* inptr pointer, which would loop through all the inodes

1.3 How will you determine all the information related to an inode?
	-by inspecting each field in the inptr we get all the relavant information about that inode

1.4 How will you determine all the blocks related to an inode?
	-we first check whether the inode is valid by inspecting the Valid field
	-we first read the direct blocks by accessing the Direct array field
	-if we also have an indirect block, we need to read the block in a new Block indirectBuffer variable
		-this should now contain a big array of block numbers which we simply traverse and print out

Brief response please!

---

2. When implementing the `format()` function, you will need to write the superblock and clear the remaining blocks in the file system.

2.1 What should happen if the the file system is already mounted?
	-we should not be formatting a mounted file system, therefore we return failure (false)
2.2 What information must be written into the superblock?
	-the superblock must have the following set
		MagicNumber 
		Blocks
		InodeBlocks
		superData
2.3 How would you clear all the remaining blocks?
	-we read the disk structure to figure out how many blocks it has
	-then starting at index 1 (index 0 is our superblock) we start clearing the remaining blocks
	-clearing is performed in the following way:
		A Block clearData variable is allocated whose bytes are all set to 0
		We use this clearData block to write to the specified block number which we want to clear
			- i.e. wwrite(blockIndex, clearData.Data);

Brief response please!

---

3. When implementing the `mount()` function, you will need to prepare a filesystem for use by reading the superblock and allocating the free block bitmap.

3.1 What should happen if the file system is already mounted?
	- we simply return failure (false)

3.2 What sanity checks must you perform before building up the free block bitmaps?
	- we need to check whether MagicNumber,	Blocks, InodeBlocks, superData fields in the superblock are properly set

3.3 How will you determine which blocks are free?
	- similarly to the debug() implementation, we would step through each valid nodes' direct and indirect blocks
	- these blocks are supposedly not free
	- as an additional note my bitmap also includes the superblock and inode blocks ... each of these is set as not free
	- NOTE: I've also implemented an inode bitmap in the Bitmap struct	

Brief response please!

---

4. To implement `create()`, you will need to locate a free inode and save a new inode into the inode table.

4.1 How will you locate a free inode?
	-a quick look in the block bitmap and using some bit manipulation can give us the first free inode in the bitmap
		i.e. getFirstFreeInode() helper does this

4.2 What information would you see in a new inode?
	-the new inode should have the Valide field set to 1, and all other fields (Size, Direct array, and Indirect) set to 0 

4.3 How will you record this new inode?
	- we first write to the inode bitmap indicating this node is now occupied
	- then from the inumber of the inode we figure out on which block the inode should be stored
		i.e. wread(inodeBlockIndex, block.Data);
	- once we obtain the block, we create an Inode* inptr to access the bytes related to the inumber of the inode
	- we then write information we would see in a new inode to this inptr
	- finally we write this updated block back to disk 
		i.e. wwrite(inodeBlockIndex, block.Data) 

Brief response please!

---

5. To implement `remove()`, you will need to locate the inode and then free its associated blocks.

5.1 How will you determine if the specified inode is valid?
	- from the inumber of the inode we figure out on which block the inode should be stored
		i.e. wread(inodeBlockIndex, block.Data);
	- once we obtain the block, we create an Inode* inptr to access the bytes related to the inumber of the inode
	- by inspecting the inumber of the inode we check whether the inode is valid

	NOTE: we could have also looked into the Inode Bitmap, however this would require less disk reads and would cause
		the test cases to fail. P.S. I've finished the implementation before the Professor said this was okay

5.2 How will you free the direct blocks?
	- similar to debug, we step through the Direct array field, and a 0 value is encountered, set the value for the
		 corresponding block number in the block bitmap to 0	

5.3 How will you free the indirect blocks?
	- similar to debug, we first read the block number specified in the Indirect field to get the list of all indirect blocks
	- next until a 0 value is encountered, set the value for the corresponding block number in the block bitmap to 0	

5.4 How will you update the inode table?
	- we write to the inptr Valid field and set it to zero
	- we then write this updated block back to disk 
		i.e. wwrite(inodeBlockIndex, block.Data);
	- NOTE: the inode Bitmap is also updated:
		setValueInInodeBitmap(inumber, 0)

Brief response please!

---

6. To implement `stat()`, you will need to locate the inode and return its size.

6.1 How will you determine if the specified inode is valid?
	- from the inumber of the inode we figure out on which block the inode should be stored
		i.e. wread(inodeBlockIndex, block.Data);
	- once we obtain the block, we create an Inode* inptr to access the bytes related to the inumber of the inode
	- by inspecting the inumber of the inode we check whether the inode is valid

6.2 How will you determine the inode's size?
	- read the Size field in the inptr

Brief response please!

---

7. To implement `read()`, you will need to locate the inode and copy data from appropriate blocks to the user-specified data buffer.

7.1  How will you determine if the specified inode is valid?
	-  from the inumber of the inode we figure out on which block the inode should be stored
		i.e. wread(inodeBlockIndex, block.Data);
	- once we obtain the block, we create an Inode* inptr to access the bytes related to the inumber of the inode
	- by inspecting the inumber of the inode we check whether the inode is valid

7.2  How will you determine which block to read from?
	- using the offset and integer division
		i.e. firstBlockIndex = offset/BLOCK_SIZE; gives the first block in the inode from which we should read from
		if firstBlockIndex <= POINTERS_PER_INODE, then we read the Direct[firstBlockIndex] block first
		otherwise we read the indirect blocks; more specifically the index of the first indrect block we should read
		from in in the list of indirect blocks is (firstBlockIndex - POINTERS_PER_INODE)
		
7.3  How will you handle the offset?
	- to correctly offset into the first block we use modular arithmetic
		i.e. byteIndex = offset % BLOCK_SIZE; 	//the byte offset inside blocks
	- however once the first block has been read the offset is reset to 0 for all subsequent blocks

7.4  How will you copy from a block to the data buffer?
	- I've decided to copy values byte by byte, each time incrementing the number of bytes read
	- once the required number of bytes is read the function returns

Brief response please!

---

8. To implement `write()`, you will need to locate the inode and copy data the user-specified data buffer to data blocks in the file system.

8.1  How will you determine if the specified inode is valid?
	-from the inumber of the inode we figure out on which block the inode should be stored
		i.e. wread(inodeBlockIndex, block.Data);
	- once we obtain the block, we create an Inode* inptr to access the bytes related to the inumber of the inode
	- by inspecting the inumber of the inode we check whether the inode is valid

8.2  How will you determine which block to write to?
	- process is similar to reading:
	- using the offset and integer division
		i.e. firstBlockIndex = offset/BLOCK_SIZE; gives the first block in the inode to which we should write to
		if firstBlockIndex <= POINTERS_PER_INODE, then we write to the Direct[firstBlockIndex] block first
		otherwise we write to the indirect blocks; more specifically the index of the first indrect block we should write
		to in in the list of indirect blocks is (firstBlockIndex - POINTERS_PER_INODE)	

8.3  How will you handle the offset?
	process is similar to reading:
	- to correctly offset into the first block we use modular arithmetic
		i.e. byteIndex = offset % BLOCK_SIZE; 	//the byte offset inside blocks
	- however once the first block has been written to the offset is reset to 0 for all subsequent blocks	

8.4  How will you know if you need a new block?
	- we consider the remaining bytes in the file starting from the offset
		i.e. remainingBytes = inptr->Size - offset;	
	- if the remaining bytes is smaller than the length argument to wwrite, we start allocating more blocks

8.5  How will you manage allocating a new block if you need another one?
	- depending on the length and remaining capacity we can figure out how many blocks we need to allocated
	- for each block we need to consider 3 cases
		1) all direct blocks are allocated but there is no indirect block 
			-allocate the indirect block
			-update the blockBitmap
			-save changes to the inode (i.e. wwrite(inodeBlockIndex, inodeBlock->Data);)

		2) there is an indirect block allocated 
			-add the new block to the list of indirect blocks
			-update the block bitmap
			-save changes to the indirect block (i.e.wwrite(inptr->Indirect, block.Data);)

		3) not all indirect blocks are allocated 
			-add the new block to the array of direct blocks
			-update the block bitmap
			-save changes to the inode (i.e. wwrite(inodeBlockIndex, inodeBlock->Data);)
	

8.6  How will you copy from a block to the data buffer?
	- I've decided to copy values and write values byte by byte, each time incrementing the number of bytes written
	- once the required number of bytes is written the function updates the inode and returns

8.7  How will you update the inode?
	- once the required number of bytes is written the inode's size is updated to reflect how many bytes were added (written)
		-NOTE: the file size may not neccesarily increase !
	- save the changes to the inode to disk (i.e. wwrite(inodeBlockIndex, inodeBlock.Data))

Brief response please!

---


Errata
------

Describe any known errors, bugs, or deviations from the requirements.

---

(Optional) Additional Test Cases
--------------------------------

Describe any new test cases that you developed.

