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

/*Prints basic info for an inode, such as inumber, size, direct blocks and indirect blocks*/
void printInode(int inumber, Inode* inptr) {
	printf("Inode %i:\n", inumber);
	printf("    size: %i bytes\n", inptr->Size);
	printf("    direct blocks:");
	int j;
	for (j = 0; j < POINTERS_PER_INODE; ++j) {
		if (inptr->Direct[j] == 0) {
			break;
		}
		printf(" %i", inptr->Direct[j]);
	}
	printf("\n");
	
	if (j == POINTERS_PER_INODE) {
		printf("    indirect block: %i\n", inptr->Indirect);
		Block indirectBuffer;
		wread(inptr->Indirect, indirectBuffer.Data);
		printf("    indirect data blocks:");
		unsigned int* indirectBlock = indirectBuffer.Pointers;						
		while (*indirectBlock != 0) {
			printf(" %i", *indirectBlock);
			++indirectBlock;
		}
		printf("\n"); 				
	}

}

/*sets values of all bytes in a block to 0*/
void formatBlock(int blockIndex) {
	Block clearData;
	memset(clearData.Data, 0, BLOCK_SIZE); 
	wwrite(blockIndex, clearData.Data);
}

/*sets bit in the BlockBitmap to the given value (0 or 1)*/
void setValueInBlockBitmap(int blockIndex, int value) {
	if (value == 0 || value == 1) {
		char mask = (0x01 << (7 - (blockIndex%8)));
		if (value) {
		bitmap.BlockBitmap[blockIndex/8] |= mask; 
		} else {
		bitmap.BlockBitmap[blockIndex/8] &= ~mask;
		}
	} else {
		printf("Cannot set value in bitmap to anything other than 0 or 1\n");
		exit(1);
	}
}

/*sets bit in the InodeBitmap to the given value (0 or 1)*/
void setValueInInodeBitmap(size_t inumber, int value) {
	if (value == 0 || value == 1) {
		char mask = (0x01 << (7 - (inumber%8)));
		if (value) {
		bitmap.InodeBitmap[inumber/8] |= mask; 
		} else {
		bitmap.InodeBitmap[inumber/8] &= ~mask;
		}
	}else {
		printf("Cannot set value in bitmap to anything other than 0 or 1\n");
		exit(1);
	}
}

/*prints both the Block and Inode Bitmaps, used when debugging*/
void printBitmaps() {
	printf("\nPrinting BlockBitmap \n");
	int byteIndex, bitIndex;
	int numBytes = ceil(((float) bitmap.BlockBitmapSize)/ 8);
	//read each byte	
	for (byteIndex = 0; byteIndex < numBytes; ++byteIndex) {
		
				char temp = bitmap.BlockBitmap[byteIndex];		//copy the byte into a temp buffer
		//calculate the index of the most significant bit that isn't 1
		for (bitIndex = 0; bitIndex < 8; ++bitIndex) {
			//we need to check whether we're at the edge of the bitmap			
			if (byteIndex*8 + bitIndex >= bitmap.BlockBitmapSize) {
				break;				
			}

			if ((temp << bitIndex) & 0x80) {	
				printf("1");
			} else {
				printf("0");
			}
		}

		printf(" ");
		if (byteIndex % 16 == 15) {
			printf("\n");
		}
	}
	printf("\n");
	printf("\nPrinting InodeBitmap \n");
	numBytes = ceil(((float) bitmap.InodeBitmapSize)/ 8);
	//read each byte	
	for (byteIndex = 0; byteIndex < numBytes; ++byteIndex) {
		char temp = bitmap.InodeBitmap[byteIndex];		//copy the byte into a temp buffer
		//calculate the index of the most significant bit that isn't 1
		for (bitIndex = 0; bitIndex < 8; ++bitIndex) {
			//we need to check whether we're at the edge of the bitmap			
			if (byteIndex*8 + bitIndex >= bitmap.InodeBitmapSize) {
				break;				
			}

			if ((temp << bitIndex) & 0x80) {	
				printf("1");
			} else {
				printf("0");
			}
		}
		printf(" ");
		if (byteIndex % 16 == 15) {
			printf("\n");
		}
	}
	printf("\n");

}

/*sanity checks performed on the super block when mounting*/
bool sanityChecks(SuperBlock* super) {
	int validMagic = super->MagicNumber == MAGIC_NUMBER;
	int validBlock = super->Blocks == disk.Blocks;
	int validInodeBlocks = super->InodeBlocks == ceil((float)(disk.Blocks) * 0.1);
	int validInodes = super->Inodes == super->InodeBlocks * (BLOCK_SIZE/sizeof(Inode));
	
	return validMagic && validBlock && validInodeBlocks && validInodes;
}

void initBitmaps(SuperBlock* super) {
	bitmap.BlockBitmapSize = super->Blocks; 		
	bitmap.InodeBitmapSize = super->Inodes; 
	bitmap.BlockBitmap = (char*) malloc((unsigned int) ceil(((float) bitmap.BlockBitmapSize)/8));	
	bitmap.InodeBitmap = (char*) malloc((unsigned int) ceil(((float) bitmap.InodeBitmapSize)/8));	

}

void mountInode(size_t inumber, Inode* inptr) {
	/*this inode is not free*/
	setValueInInodeBitmap(inumber, 1);
	//for each valid inode, check all direct and indirect blocks and mark them as not free
	int j;
	for (j = 0; j < POINTERS_PER_INODE; ++j) {
		if (inptr->Direct[j] == 0) {
			break;
		}
		//set the value of the direct block in bitmap as not free
		setValueInBlockBitmap(inptr->Direct[j], 1);
	}
			
	if (j == POINTERS_PER_INODE && inptr->Indirect) {
		setValueInBlockBitmap(inptr->Indirect, 1);
		Block indirectBuffer;
		wread(inptr->Indirect, indirectBuffer.Data);
		unsigned int* indirectBlock = indirectBuffer.Pointers;					
		while (*indirectBlock != 0) {
			setValueInBlockBitmap(*indirectBlock, 1);
			++indirectBlock;
		}	
	}
}

/*returns the first block# that is free in the bitmap*/
/*returns -1 if there are no free blocks*/
unsigned int getFirstFreeBlock() {
	int byteIndex, bitIndex;
	int numBytes = ceil(((float) bitmap.BlockBitmapSize)/ 8);	//calculate the number of bytes in the bitmap
	//read each byte	
	for (byteIndex = 0; byteIndex < numBytes; ++byteIndex) {
		char temp = bitmap.BlockBitmap[byteIndex];		//copy the byte into a temp buffer
		//calculate the index of the most significant bit that isn't 1
		for (bitIndex = 0; bitIndex < 8; ++bitIndex) {
			//we need to check whether we're at the edge of the bitmap			
			if (byteIndex*8 + bitIndex >= bitmap.BlockBitmapSize) {
				return -1;				
			}
			//if there's at least 1 bit that isn't 1, we have our free block, and we return its index		 
			if (!((temp << bitIndex) & 0x80)) {	
				return byteIndex * 8 + bitIndex;
			} 
		}
	}
	return -1;

}


/*returns the inode number of the first free node it encounters in the InodeBitmap*/
/*if no free nodes exist -1 is returned*/
ssize_t getFirstFreeInode() {
	int byteIndex, bitIndex;
	int numBytes = ceil(((float) bitmap.InodeBitmapSize)/ 8);	//calculate the number of bytes in the bitmap
	//read each byte	
	for (byteIndex = 0; byteIndex < numBytes; ++byteIndex) {
		char temp = bitmap.InodeBitmap[byteIndex];		//copy the byte into a temp buffer
		//calculate the index of the most significant bit that isn't 1
		for (bitIndex = 0; bitIndex < 8; ++bitIndex) {
			//we need to check whether we're at the edge of the bitmap			
			if (byteIndex*8 + bitIndex >= bitmap.InodeBitmapSize) {
				return -1;				
			}
			//if there's at least 1 bit that isn't 1, we have our free block, and we return its index		 
			if (!((temp << bitIndex) & 0x80)) {	
				return byteIndex * 8 + bitIndex;
			} 
		}
	}
	return -1;

}

/*returns true if inumber is Valid, returns false otherwise*/
bool isInumberValid(size_t inumber) {
	int inodes = bitmap.InodeBitmapSize;
	/*check whether in bounds*/
	if(inumber < 0 || inumber >= inodes) {
		return -1;
	} 

	//based on the inumber we calculate the byteIndex and bitIndex
	int byteIndex = inumber/8;
	int bitIndex = inumber%8;
	
	/*obtain the relevant byte*/
	char temp = bitmap.InodeBitmap[byteIndex];
	
	/*if the relevant bit is 1, return true*/
	if ((temp << bitIndex) & 0x80) {
		return true;
	}

	return false;

}

/*based on the inumber, returns the block in which the the node with that inumber is located*/
unsigned int getBlockIndexFromInodeNumber(size_t inumber) {
	return 1 + (inumber / INODES_PER_BLOCK);
}

//checks all direct and indirect blocks of an inode and mark them as free in the BlockBitmap
extern void freeInode(Inode* inptr) {
	int j;
	for (j = 0; j < POINTERS_PER_INODE; ++j) {
		if (inptr->Direct[j] == 0) {
			break;
		}
		setValueInBlockBitmap(j, 0);
	}
		
	if (j == POINTERS_PER_INODE && inptr->Indirect) {
		Block indirectBuffer;
		wread(inptr->Indirect, indirectBuffer.Data);
		unsigned int* indirectBlock = indirectBuffer.Pointers;					
		while (*indirectBlock != 0) {
			setValueInBlockBitmap(*indirectBlock, 0);
			++indirectBlock;
		}	
	}			
	inptr->Valid = 0;

}

/*allocates however many blocks are necessary to an inode when writing to it*/
/*if it needs to allocate 8 inodes, and there are only 5, it will allocate 5*/
extern void allocateMoreBlocks(Block* inodeBlock, size_t inumber, size_t length, size_t offset, int* bytesToWrite) {

	Inode* inptr = (Inode*) (inodeBlock->Data + INODE_SIZE * inumber); /*create a pointer to the inode*/
	int inodeBlockIndex = getBlockIndexFromInodeNumber(inumber); 	/*get the blockIndex where the inode is located*/

	/*if we start from the offset and continue, how many bytes does the remainder of the file contain*/
	int remainingCapacity = inptr->Size - offset;
	/*how many blocks need to get allocated*/					
	int blocksToAllocate = ceil(((float)(length - remainingCapacity))/BLOCK_SIZE);

	/*attemp allocating the blocks*/
	for (int i = 0; i < blocksToAllocate; ++i) {

		/* 1) The Direct Pointers are Full and There is No Indirect Block */
		
		if (!inptr->Indirect && inptr->Direct[POINTERS_PER_INODE - 1]){
			inptr->Indirect = getFirstFreeBlock();		//allocate an indirect block for the inode
			setValueInBlockBitmap(inptr->Indirect, 1); 	//set block as taken in bitmap			
			formatBlock(inptr->Indirect);			//format this newly allocated block
			wwrite(inodeBlockIndex, inodeBlock->Data);	//save changes to the inode to disk
		}

		unsigned int newBlock = getFirstFreeBlock();	/*this new block will be allocated to the inode*/

		/* 2) There Is an Indirect Block*/
		if (inptr->Indirect) {

			Block block;					/*generic block for reading indirect block list*/
			wread(inptr->Indirect, block.Data);		/*read the indirect block*/

			/*maybe there are no more free blocks*/
			if (newBlock == -1) {
				*bytesToWrite = i * BLOCK_SIZE + remainingCapacity;	/*update how many bytes we can write*/
				break;			/*break out of the for loop which makes the function return*/		
			}
			/*loop through the blocks and find the index where we can insert the new block*/
			int j;
			for(j = 0; j < POINTERS_PER_BLOCK; ++j) {
				if (block.Pointers[j] == 0) {
					break;
				}
			}

			/*maybe no more blocks can be added to list of indirect blocks*/
			if (j == POINTERS_PER_BLOCK) {
				*bytesToWrite = i * BLOCK_SIZE + remainingCapacity;	/*update how many bytes we can write*/
				break;		/*break out of the for loop which makes the function return*/
			}
			
			block.Pointers[j] = newBlock;		//update the data in the generic block
			setValueInBlockBitmap(newBlock, 1); 	//set block as taken in bitmap
			wwrite(inptr->Indirect, block.Data);	//save the change to disk
		}
		/* 3) There are only Direct Blocks*/
		else {
			/*maybe there are no more free blocks*/
			if (newBlock == -1) {
				printf("No more free blocks available, I allocated however many I could\n");
				*bytesToWrite = i * BLOCK_SIZE + remainingCapacity; /*update how many bytes we can write*/
				break;
			}
			/*find the appropriate spot to insert the new block into*/
			for(int j = 0; j < POINTERS_PER_INODE; ++j) {
				if (inptr->Direct[j] == 0) {			//once that spot is found
					inptr->Direct[j] = newBlock;		//update the data in the inode
					setValueInBlockBitmap(newBlock, 1); 	//set block as taken in bitmap
					wwrite(inodeBlockIndex, inodeBlock->Data);//save the changes to disk
					break;
				}
			}
		}
	}
}

/*returns the total increase to file size after writing to a file*/
unsigned int increaseToFileSize(unsigned int remainingCapacity, unsigned int bytesToWrite) {
	if (remainingCapacity > bytesToWrite) {
		return 0;	/*the file size doesn't have to change, we are simply overwriting bytes*/
	} else {
		return bytesToWrite - remainingCapacity;	/*otherwise the file gets extended by this many bytes*/
	}

}
