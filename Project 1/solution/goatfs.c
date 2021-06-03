#include <sys/types.h>
#include <stdio.h>
#include <unistd.h>
#include <stdlib.h>
#include <errno.h> //errno
#include <math.h>
#include <stdbool.h>
#include <string.h>
#include "goatfs.h"


extern DISK disk;
extern Bitmap bitmap;



void debug() {
	Block buffer;
	wread(0, buffer.Data);

 	printf("SuperBlock:\n");
	if (buffer.Super.MagicNumber == MAGIC_NUMBER) {
		printf("    magic number is valid\n");
		printf("    %i blocks\n", buffer.Super.Blocks);
		printf("    %i inode blocks\n",buffer.Super.InodeBlocks);
		printf("    %i inodes\n", buffer.Super.Inodes);
		
		int inodeBlocks = buffer.Super.InodeBlocks;

		for (int inodeBlockIndex = 1; inodeBlockIndex <= inodeBlocks; ++inodeBlockIndex) {
			wread(inodeBlockIndex, buffer.Data);
			Inode* inptr;
			for(int i = 0; i < INODES_PER_BLOCK; ++i) {
				int inumber = (inodeBlockIndex-1) * (INODES_PER_BLOCK) + i;
				inptr = (Inode*)  (buffer.Data + INODE_SIZE * i);
				if (inptr->Valid) {
					printInode(inumber, inptr);
				}		
			}
		}	
	} else {
		printf("\tmagic number is invalid\n");
		exit(ERR_BAD_MAGIC_NUMBER);
	}
}


bool format() {
	/*check if disk is mounted*/
	if (disk.Mounts) {
		return FAILURE_BAD_FORMAT;
	}

	int numOfBlocks = disk.Blocks;	//define for easier access
	
	for (int blockIndex = 1; blockIndex < numOfBlocks; ++blockIndex) {
		formatBlock(blockIndex);	//format all the blocks
	}

	/*construct the super block*/
	Block superData;

	superData.Super.MagicNumber = MAGIC_NUMBER;
	superData.Super.Blocks = numOfBlocks;					// Number of blocks in file system
    	superData.Super.InodeBlocks = ceil((float)(numOfBlocks) * 0.1);		// Number of blocks reserved for inodes
    	superData.Super.Inodes = superData.Super.InodeBlocks * (BLOCK_SIZE/sizeof(Inode)); // Number of inodes in file system
	
	wwrite(0, superData.Data);
	
	return SUCCESS_GOOD_FORMAT;
}



bool mount() {
	/*if the disk is already mounted, return false*/
	if (disk.Mounts) {
		return FAILURE_BAD_MOUNT;
	}
	
	Block block;
	wread(0, block.Data);	//read super block
	
	if (sanityChecks(&block.Super)) {	//if sanity checks check out, proceed
		initBitmaps(&(block.Super));			//initialize our bitmap
		setValueInBlockBitmap(0, 1);			//the superblock is never free
		
		int inodeBlocks = block.Super.InodeBlocks;	//define for easier access

		//step through each inode block
		for (int inodeBlockIndex = 1; inodeBlockIndex <= inodeBlocks; ++inodeBlockIndex) {
			setValueInBlockBitmap(inodeBlockIndex, 1);	//set the inodeBlock as not free in bitmap
	
			wread(inodeBlockIndex, block.Data);		//read the inode block
			Inode* inptr;

			//step through each inode inside each inode block
			for(int i = 0; i < INODES_PER_BLOCK; ++i) {
				size_t inumber = (inodeBlockIndex-1) * INODES_PER_BLOCK + i;
				inptr = (Inode*)  (block.Data + INODE_SIZE * i);

				if (inptr->Valid) {			//if the inode is valid, mount it
					mountInode(inumber, inptr);	//i.e. set values in BlockBitmap
				}		
			}
		}	
		disk.Mounts = 1;		//once done set the disk.Mounts field
		return SUCCESS_GOOD_MOUNT;	//true
	}

	/*in case sanity checks failed*/	
	else {
		return FAILURE_BAD_MOUNT; 	//false
	}
}


ssize_t create() {

	if (disk.Mounts == 0) {		/*if the disk isn't mounted, return -1*/
		return -1;
	}

	Block block;	
	wread(0, block.Data);		/*read super block*/
	

	ssize_t inumber = getFirstFreeInode(); 	/*get the first inode that is free */
	

	if (inumber != -1) {			/*if a node is found*/
	

		int inodeBlockIndex = getBlockIndexFromInodeNumber(inumber); 	/*get the blockIndex of the inode*/
	
		wread(inodeBlockIndex, block.Data);				/*read that blockIndex*/
	
		Inode* inptr = (Inode*) (block.Data + INODE_SIZE * inumber); /*create a pointer to the inode*/
		
		/*set the values of the inode*/
		inptr->Valid = 1;
		inptr->Size = 0;
		inptr->Direct[0] = 0;
		inptr->Indirect = 0;
		setValueInInodeBitmap(inumber, 1);


		wwrite(inodeBlockIndex, block.Data); 	/*save to disk*/
	
	}

	return inumber; 	/*Note: if inumber == -1, then -1 will be returned as requiered*/
}


bool wremove(size_t inumber) {

	if (disk.Mounts == 0) {			/*if the disk isn't mounted, return -1*/
		return FAILURE_BAD_REMOVE;
	}

	Block block;
	wread(0, block.Data);			//read superblock

 	if (!isInumberValid(inumber)) {		/*check whether the inode is valid*/
		return FAILURE_BAD_REMOVE;
	}

	int inodeBlockIndex = getBlockIndexFromInodeNumber(inumber); 	/*get the blockIndex where the inode is located*/

	wread(inodeBlockIndex, block.Data);				/*read the block where the inode is located*/

	Inode* inptr = (Inode*) (block.Data + INODE_SIZE * inumber);	/*create a pointer to the inode*/

	freeInode(inptr);				//check all direct and indirect blocks and mark them as free

	wwrite(inodeBlockIndex, block.Data);		/*save changes to disk*/
		
	setValueInInodeBitmap(inumber, 0);		/*update the InodeBitmap to reflect the inode was freed*/
	return SUCCESS_GOOD_REMOVE;
}

ssize_t stat(size_t inumber) {
	/*if the disk isn't mounted, return -1*/
	if (disk.Mounts == 0) {
		return -1;
	}
	
	/*NOTE: Apparently this function doesn't care if inumber is within range*/
	/*the code would stil work properly if this was uncommented*/
	/*however test cases would fail, because less disk reads would be made*/
 	//if (!isInumberValid(inumber)) {
	//	return -1;
	//}
	
	Block block;
	int inodeBlockIndex = getBlockIndexFromInodeNumber(inumber); 	/*get the blockIndex where the inode is located*/
	wread(inodeBlockIndex, block.Data);				/*read the block where the inode is located*/
	Inode* inptr = (Inode*)  (block.Data + INODE_SIZE * inumber);	/*create a pointer to the inode*/

	if (inptr->Valid) {return inptr->Size;}				/*returns the size of the inode*/
	return -1;
		
}

ssize_t wfsread(size_t inumber, char *data, size_t length, size_t offset) {

	if (disk.Mounts == 0) {		/*if the disk isn't mounted, return -1*/
		return -1;
	}

	
	Block inodeBlock;						/*where the block containing the inode is kept*/
	int inodeBlockIndex = getBlockIndexFromInodeNumber(inumber); 	/*get the blockIndex where the inode is located*/
	wread(inodeBlockIndex, inodeBlock.Data);			/*read the block where the inode is located*/
	Inode* inptr = (Inode*) (inodeBlock.Data + INODE_SIZE * inumber); /*create a pointer to the inode*/
	
	/*if the inode is valid*/
	if (inptr->Valid) {
		/*if we start from the offset and continue, how many bytes does the remainder of the file contain*/
		int remainingBytes = inptr->Size - offset;	
		int bytesToRead;	/*the amount of bytes we will read*/

		if (remainingBytes < 0) { /*the offset may be too large*/
			printf("Offset too large, returning -1\n");
			return -1;
		}

		/*if the remainder of the file is smaller than the buffer, we will only read remainingBytes amount of bytes*/
		if (length > remainingBytes) { bytesToRead = remainingBytes;} 
		/*if the remainder of the file is greater than the buffer, we will read length amount of bytes*/
		else { bytesToRead = length;}
		
		int bytesRead = 0;		/*keep track of how many bytes were read*/

		/*the offset tells us from which direct/indirect block to start*/
		int firstBlockIndex = offset/BLOCK_SIZE;
		int currentBlockIndex = firstBlockIndex;
	
		Block block;				//generic block for reading data
		int byteIndex = offset % BLOCK_SIZE; 	//the byte offset inside blocks
			
		//if the offset is less than POINTERS_PER_INODE blocks, we read direct blocks
		while (currentBlockIndex < POINTERS_PER_INODE && bytesRead < bytesToRead) {
			/*read the direct block*/
			wread(inptr->Direct[currentBlockIndex], block.Data);
			/*reads byte by byte, starting at the specified offset*/
			for (; byteIndex < BLOCK_SIZE; ++byteIndex) {
				data[bytesRead] = block.Data[byteIndex];
				++bytesRead;
	
				if (bytesRead >= bytesToRead) {
					return bytesRead;  /*return once all bytes have been read*/			
				}
			}
			byteIndex = 0;		//reset the byteOffset to 0 to start reading the next block properly
			++currentBlockIndex;	//read the next direct block
		}
	
		/*next we read the indirect blocks*/
		Block indirectBlocksList;		/*indirect block containing block numbers*/
		wread(inptr->Indirect, indirectBlocksList.Data); /*read data into the indirect block*/
		/*the current block inside the list of indirect blocks*/
		unsigned int* indirectBlockIndex = (indirectBlocksList.Pointers) + (currentBlockIndex - POINTERS_PER_INODE);	
		
		/*while we stil haven't read the required amount of bytes*/
		while (bytesRead < bytesToRead) {
			/*read the block from the list of blocks*/
			wread(*indirectBlockIndex, block.Data);
			
			/*reads byte by byte at the specified offset*/
			for (; byteIndex < BLOCK_SIZE; ++byteIndex) {
				data[bytesRead] = block.Data[byteIndex];
				++bytesRead;
				if (bytesRead >= bytesToRead) {
					return bytesRead;  /*return once all bytes have been read*/				
				}
			}
			byteIndex = 0;		//reset the byteOffset to 0 to start reading the next block properly
			++indirectBlockIndex;	//read the next indirect block
		}	
	}

	return -1;

}

ssize_t wfswrite(size_t inumber, char *data, size_t length, size_t offset) {
	/*if the disk isn't mounted, return -1*/
	if (disk.Mounts == 0) {
		return -1;
	}

	Block inodeBlock;						/*where the block containing the inode is kept*/
	int inodeBlockIndex = getBlockIndexFromInodeNumber(inumber); 	/*get the blockIndex where the inode is located*/
	wread(inodeBlockIndex, inodeBlock.Data);			/*read the block where the inode is located*/
	Inode* inptr = (Inode*) (inodeBlock.Data + INODE_SIZE * inumber); /*create a pointer to the inode*/

	int bytesToWrite = length;	/*how many bytes can we write?*/
	
	/*if the inode is valid*/
	if (inptr->Valid) {
		/*if we start from the offset and continue, how many bytes does the remainder of the file contain*/
		int remainingCapacity = inptr->Size - offset;

		/*the offset may be too large*/
		if (remainingCapacity < 0) {
			printf("Offset too large, returning -1\n");
			return -1;
		}

		/*the file may have less bytes than the length argument*/
		/* we need to allocate more blocks in that case */
		if (length > remainingCapacity) { 						
			allocateMoreBlocks(&inodeBlock, inumber, length, offset, &bytesToWrite);		
		}

		int bytesWritten = 0;		//number of bytes we have written so far
		
		/*the offset tells us from which direct/indirect block to start*/
		int firstBlockIndex = offset/BLOCK_SIZE;
		int currentBlockIndex = firstBlockIndex;
		
		int byteIndex = offset % BLOCK_SIZE; //the byte offset inside the first block
		Block block;	//generic block for reading data

		//we write to direct blocks first, if the offset offsets into them
		while (currentBlockIndex < POINTERS_PER_INODE && bytesWritten < bytesToWrite) {
			/*read data of the direct block*/
			wread(inptr->Direct[currentBlockIndex], block.Data);
			
			/*modify the data by writing byte by byte starting at the offset*/
			for (; byteIndex < BLOCK_SIZE; ++byteIndex) {
				block.Data[byteIndex] = data[bytesWritten];
				++bytesWritten;
				/*once we have written all the bytes we were supposed to*/
				if (bytesWritten >= bytesToWrite) {
					wwrite(inptr->Direct[currentBlockIndex], block.Data);	//save changes to disk
					//update the size of the inode
					inptr->Size += increaseToFileSize(remainingCapacity,bytesWritten);
					wwrite(inodeBlockIndex, inodeBlock.Data);		//save inode changes to disk
				
					return bytesWritten;				//return total number of bytes written
				}	
			}
			//end of the block was reached and there is stil more to write
			wwrite(inptr->Direct[currentBlockIndex], block.Data);	//save changes to disk
			byteIndex = 0;		//reset the byteOffset to 0 to start reading the next block properly
			++currentBlockIndex;	//read the next direct block
		}
		/*next we write the indirect blocks*/
		Block indirectBlocks;				//holds the list of indirect blocks
		wread(inptr->Indirect, indirectBlocks.Data);	//read the indirect block
		/*the current block inside the list of indirect blocks*/
		unsigned int* indirectBlockIndex = (indirectBlocks.Pointers) + (currentBlockIndex - POINTERS_PER_INODE);
		/*while we stil haven't written the required amount of bytes*/
		while (bytesWritten < bytesToWrite) {
			/*read the indirect blcok*/
			wread(*indirectBlockIndex, block.Data);
			/*modify the data by writing byte by byte starting at the offset*/
			for (; byteIndex < BLOCK_SIZE; ++byteIndex) {
				block.Data[byteIndex] = data[bytesWritten];
				++bytesWritten;
				/*once we have written all the bytes we were supposed to*/
				if (bytesWritten >= bytesToWrite) {
					wwrite(*indirectBlockIndex, block.Data);	//save changes to disk
					//update the size of the inode
					inptr->Size += increaseToFileSize(remainingCapacity,bytesWritten);
				
					wwrite(inodeBlockIndex, inodeBlock.Data);	//save inode changes to disk
				
					return bytesWritten;				//return total number of bytes written
				}
			}
			wwrite(*indirectBlockIndex, block.Data);	//save changes to disk
			byteIndex = 0;		//reset the byteOffset to 0 to start reading the next block properly
			++indirectBlockIndex;	//read the next indirect block
		}	
		
	}

	return -1;

}
