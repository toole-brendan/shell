# Edits for rpcserver.go

This file outlines the necessary edits to fix the linter errors in `rpcserver.go`. The errors stem from two main issues: type mismatches between local `shell` packages and `btcsuite/btcd` packages, and API changes in the `mempool` package.

## 1. Import Changes

Ensure the `internal/convert` package is imported:

```go
import (
	// ... other imports
	"github.com/toole-brendan/shell/internal/convert"
	// ... other imports
)
```

## 2. Struct Definition Change

In the `rpcserverConfig` struct definition (around line 4867), change the type of the `TxMemPool` field to the concrete type `*mempool.TxPool`.

-   **Locate:** `rpcserverConfig` struct.
-   **Change this line:**
    ```go
    TxMemPool mempool.TxMempool
    ```
-   **To this:**
    ```go
    TxMemPool *mempool.TxPool
    ```

## 3. Function Body Changes

The following are specific line-by-line changes needed within various functions in `rpcserver.go`.

### In `handleCreateRawTransaction`:

-   **Around line 568:** Use the conversion function for decoding addresses.
    ```go
    // FROM:
    addr, err := btcutil.DecodeAddress(encodedAddr, params)

    // TO:
    addr, err := btcutil.DecodeAddress(encodedAddr, convert.ParamsToBtc(params.Name))
    ```
-   **Around line 588:** Use the conversion function for the network check.
    ```go
    // FROM:
    if !addr.IsForNet(params) {
    
    // TO:
    if !addr.IsForNet(convert.ParamsToBtc(params.Name)) {
    ```

### In `handleDecodeScript`:

-   **Around line 777:** Use the conversion function for `NewAddressScriptHash`.
    ```go
    // FROM:
    p2sh, err := btcutil.NewAddressScriptHash(script, s.cfg.ChainParams)

    // TO:
    p2sh, err := btcutil.NewAddressScriptHash(script, convert.ParamsToBtc(s.cfg.ChainParams.Name))
    ```

### In `createTxRawResult`:

-   **Around line 818:** Wrap the transaction with `convert.NewShellTx`.
    ```go
    // FROM:
    Vsize:    int32(mempool.GetTxVirtualSize(mtx)),
    Weight:   int32(blockchain.GetTransactionWeight(mtx)),
    
    // TO:
    Vsize:    int32(mempool.GetTxVirtualSize(convert.NewShellTx(mtx))),
    Weight:   int32(blockchain.GetTransactionWeight(convert.NewShellTx(mtx))),
    ```

### In `handleGetBlock`:

-   **Around line 1133:** Wrap the block with `convert.NewShellBlockFromBtcBlock`.
    ```go
    // FROM:
    Weight:        int32(blockchain.GetBlockWeight(blk)),

    // TO:
    Weight:        int32(blockchain.GetBlockWeight(convert.NewShellBlockFromBtcBlock(blk))),
    ```
-   **Around line 1161:** Convert `btcd` types to local `shell` types.
    ```go
    // FROM:
    rawTxn, err := createTxRawResult(params, tx.MsgTx(),
        tx.Hash().String(), blockHeader, hash.String(),
        blockHeight, best.Height)

    // TO:
    rawTxn, err := createTxRawResult(params, convert.ToShellMsgTx(tx.MsgTx()),
        tx.Hash().String(), convert.ToShellBlockHeader(&blk.MsgBlock().Header), hash.String(),
        blockHeight, best.Height)
    ```

### In `handleGetBlockTemplateProposal`:

-   **Around line 2156:** Convert the `btcd` hash to a `shell` hash for comparison.
    ```go
    // FROM:
    if !expectedPrevHash.IsEqual(prevHash) {

    // TO:
    if !expectedPrevHash.IsEqual(convert.HashToShell(prevHash)) {
    ```

### In `handleGetMiningInfo`:

-   **Around line 2417:** Fix the type cast for `HashesPerSec`.
    ```go
    // FROM:
    HashesPerSec:       int64(s.cfg.CPUMiner.HashesPerSecond()),

    // TO:
    HashesPerSec:       s.cfg.CPUMiner.HashesPerSecond(),
    ```

### In `handleSendRawTransaction`:

-   **Around line 3453:** Add the `mempool.Tag` type cast.
    ```go
    // FROM:
    acceptedTxs, err := s.cfg.TxMemPool.ProcessTransaction(tx, false, false, 0)

    // TO:
    acceptedTxs, err := s.cfg.TxMemPool.ProcessTransaction(tx, false, false, mempool.Tag(0))
    ```
-   **Around line 3506 & 3507:** Correct the comparison and argument for `RemoveTransaction`.
    ```go
    // FROM:
    if len(acceptedTxs) == 0 || !acceptedTxs[0].Tx.Hash().IsEqual(convert.HashToShell(tx.Hash())) {
        s.cfg.TxMemPool.RemoveTransaction(tx, true)

    // TO:
    if len(acceptedTxs) == 0 || !acceptedTxs[0].Tx.Hash().IsEqual(tx.Hash()) {
		s.cfg.TxMemPool.RemoveTransaction(acceptedTxs[0].Tx, true)
    ```

### In `handleSignMessageWithPrivKey`:

-   **Around line 3590:** Use the conversion function for the network check.
    ```go
    // FROM:
    if !wif.IsForNet(s.cfg.ChainParams) {

    // TO:
    if !wif.IsForNet(convert.ParamsToBtc(s.cfg.ChainParams.Name)) {
    ```

### In `handleValidateAddress`:

-   **Around line 3659:** Use the conversion function to decode the address.
    ```go
    // FROM:
    addr, err := btcutil.DecodeAddress(c.Address, s.cfg.ChainParams)

    // TO:
    addr, err := btcutil.DecodeAddress(c.Address, convert.ParamsToBtc(s.cfg.ChainParams.Name))
    ```

### In `handleVerifyMessage`:

-   **Around line 3759:** Use the conversion function for decoding the address.
    ```go
    // FROM:
    addr, err := btcutil.DecodeAddress(c.Address, params)

    // TO:
    addr, err := btcutil.DecodeAddress(c.Address, convert.ParamsToBtc(params.Name))
    ```
-   **Around line 3805:** Pass the network name string to `ParamsToBtc`.
    ```go
    // FROM:
    address, err := btcutil.NewAddressPubKey(serializedPK, convert.ParamsToBtc(params))

    // TO:
    address, err := btcutil.NewAddressPubKey(serializedPK, convert.ParamsToBtc(params.Name))
    ```
### In `handleTestMempoolAccept`:

-   **Around line 3867:** Wrap the `tx` with `convert.NewShellTx`.
    ```go
    // FROM:
    result, err := s.cfg.TxMemPool.CheckMempoolAcceptance(tx)
    // TO:
    result, err := s.cfg.TxMemPool.CheckMempoolAcceptance(convert.NewShellTx(tx))
    ```

### In `handleSearchRawTransactions`:

-   **Around line 3198:**
    ```go
    // FROM:
    addr, err := btcutil.DecodeAddress(c.Address, params)
    // TO:
    addr, err := btcutil.DecodeAddress(c.Address, convert.ParamsToBtc(params.Name))
    ```
-   **Around line 3332:**
    ```go
    // FROM:
    hexTxns[i], err = messageToHex(rtx.tx.MsgTx())
    // TO:
    hexTxns[i], err = messageToHex(convert.ToShellMsgTx(rtx.tx.MsgTx()))
    ```
-   **Around line 3372:**
    ```go
    // FROM:
    mtx = rtx.tx.MsgTx()
    // TO:
    mtx = convert.ToShellMsgTx(rtx.tx.MsgTx())
    ```

</rewritten_file> 