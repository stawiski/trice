/*! \file trice.c
\brief trices to transfer buffer functionality
\details The trices are dumped as 32bit values into a 32 bit fifo.
That is the time critical part. 
\li a trice is a 16 bit ID with a 16 bit data value
\li trices with more data are split into several 32bit basic trice values with IDs = 0 in front.
\li So a basic trice (subtrace) consists always of 4 byte.
\li for trice transmission each basic trice gets a header of additional 4 bytes.
\li The header contains a startbyte, client and server address and a crc8 exOr checksum.
\author thomas.hoehenleitner [at] seerose.net
*******************************************************************************/

#include "trice.h"
#include "treyferCrypto.h"

#ifdef TRICE_PRINTF_ADAPTER
#include <stdarg.h>
#include "printf.h"
#endif // #ifdef TRICE_PRINTF_ADAPTER

#ifdef TRICE_OFF

/*! This function should be called inside the transmit done device interrupt.
Also it should be called cyclically to trigger transmission start.
*/
void triceTxHandler( int* pTxState ){}

#else // #ifdef TRICE_OFF

#include <stdint.h>

//! trice fifo instance
ALIGN4 uint32_t triceFifo[ TRICE_FIFO_SIZE>>2 ] ALIGN4_END;

uint32_t rdIndexTriceFifo = 0; //!< trice fifo read index
uint32_t wrIndexTriceFifo = 0; //!< trice fifo write index

//! partial prefilled trice message transmit buffer 
ALIGN4 static triceMsg_t triceMsg ALIGN4_END = {
    { TRICE_START_BYTE,  TRICE_LOCAL_ADDR,  TRICE_DISPL_ADDR, 0 }, // crc8
    { 0, 0 } // 16bit ID, 16bit data
};

static uint8_t const * const limit = (uint8_t*)(&triceMsg + 1); //!< trice message buffer limit
static uint8_t       *       pRead = (uint8_t*)(&triceMsg + 1); //!< trice message buffer read pointer

/*! get next trice byte for transmission from trice message buffer, no depth check here
\retval data byte
*/
TRICE_INLINE uint8_t triceMsgNextByte( void ){
    return *pRead++;
}

/*! return count of bytes ready for transmission
\retval count
*/
static size_t triceMsgDepth( void ){
    size_t count = limit - pRead;
    if( count ){
        return count;
    } else {
        if( triceFifoDepth() ){
            triceFifoPop( (uint32_t*)(&(triceMsg.ld)) );
            pRead = (uint8_t*)&triceMsg;
            triceMsg.hd.crc8  = TRICE_START_BYTE ^ TRICE_LOCAL_ADDR ^ TRICE_DISPL_ADDR
                                 ^ triceMsg.ld.load[0]
                                 ^ triceMsg.ld.load[1]
                                 ^ triceMsg.ld.load[2]
                                 ^ triceMsg.ld.load[3];
            #ifdef TREYFER_KEY
            {
                static uint8_t const treyferKey[8] = TREYFER_KEY;
                encrypt( (uint8_t*)&triceMsg, treyferKey );
            }
            #endif // #ifdef TREYFER_KEY
            return 8;
        }
    }
    return 0;
}

/*! This function should be called inside the transmit done device interrupt.
Also it should be called cyclically to trigger transmission start.
\param pTxState address of a transmission state variable. It is cleared if no more traceLog messages to transmit and set to 1 if a traceLog transmission was started.
\todo handle 8==traceLogMsgDepth() to give chance to other data streams
*/
void triceTxHandler( int* pTxState ){
    if( triceTxDataRegisterEmpty() ){ 
        if( triceMsgDepth() ){
            uint8_t x = triceMsgNextByte();
            triceTransmitData8( x );
            *pTxState = 1;
            triceEableTxEmptyInterrupt(); 
        }else{
            triceDisableTxEmptyInterrupt();
            *pTxState = 0;
        }
    }
}

#ifdef TRICE_PRINTF_ADAPTER

//! trice replacement helper for printf() with %s 
//! use only for dynamic generatd strings
//! \param pFmt formatstring
int tricePrintfAdapter( const char* pFmt, ... ){
    va_list va;
    int done;
    static char buffer[TRICE_PRINTF_ADAPTER_BUFFERSIZE];
    
    va_start (va, pFmt );
    done = vsnprintf( buffer, sizeof(buffer), pFmt, va);
    va_end (va);

    triceString( 0, buffer );
    return done;
}

//! unused dummy definition for linker
void _putchar(char character){
}

#endif // #ifdef TRICE_PRINTF_ADAPTER

#endif // #else // #ifdef TRICE_OFF
