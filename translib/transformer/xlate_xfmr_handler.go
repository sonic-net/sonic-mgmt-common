////////////////////////////////////////////////////////////////////////////////
//                                                                            //
//  Copyright 2020 Dell, Inc.                                                 //
//                                                                            //
//  Licensed under the Apache License, Version 2.0 (the "License");           //
//  you may not use this file except in compliance with the License.          //
//  You may obtain a copy of the License at                                   //
//                                                                            //
//  http://www.apache.org/licenses/LICENSE-2.0                                //
//                                                                            //
//  Unless required by applicable law or agreed to in writing, software       //
//  distributed under the License is distributed on an "AS IS" BASIS,         //
//  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.  //
//  See the License for the specific language governing permissions and       //
//  limitations under the License.                                            //
//                                                                            //
////////////////////////////////////////////////////////////////////////////////

package transformer

import (
	"github.com/Azure/sonic-mgmt-common/translib/db"
	log "github.com/golang/glog"
)

func xfmrHandlerFunc(inParams XfmrParams, xfmrFuncNm string, ygotCtx *ygotUnMarshalCtx) error {
	const DBTY_SBT_XFMR_RET_ERR_INDX = 0
	if inParams.uri != inParams.requestUri {
		yerr := ygotXlator{ygotCtx}.translate()
		if yerr != nil {
			xfmrLogDebug("Failed to generate the ygot Node for uri(\"%v\") err(%v).", inParams.uri, yerr)
			return yerr
		}
	}

	inParams.pruneDone = new(bool)
	xfmrLogDebug("Before calling dbToYang subtree xfmr %v, inParams %v", xfmrFuncNm, inParams)
	ret, err := XlateFuncCall(dbToYangXfmrFunc(xfmrFuncNm), inParams)
	xfmrLogDebug("After calling dbToYang subtree xfmr %v, inParams %v", xfmrFuncNm, inParams)
	if err != nil {
		xfmrLogDebug("Failed to retrieve data for xpath(\"%v\") err(%v).", inParams.uri, err)
		return err
	}
	if (ret != nil) && (len(ret) > 0) {
		// db to YANG subtree xfmr returns err as the only value in return data list from <xfmr_func>.Call()
		if ret[DBTY_SBT_XFMR_RET_ERR_INDX].Interface() != nil {
			err = ret[DBTY_SBT_XFMR_RET_ERR_INDX].Interface().(error)
			if err != nil {
				log.Warningf("xfmr function(\"%v\") returned error - %v.", xfmrFuncNm, err)
			}
		}
	}
	if (err == nil) && inParams.queryParams.isEnabled() && !(*inParams.pruneDone) {
		log.Infof("xfmrPruneQP: func %v URI %v, requestUri %v",
			xfmrFuncNm, inParams.uri, inParams.requestUri)
		err = xfmrPruneQP(inParams.ygRoot, inParams.queryParams,
			inParams.uri, inParams.requestUri, inParams.ctxt)
		if err != nil && !isReqContextCancelledError(err) {
			xfmrLogInfo("xfmrPruneQP: returned error %v", err)
			// following will allow xfmr to distinguish subtree vs pruning API err to abort GET request
			err = &qpSubtreePruningErr{subtreePath: inParams.uri}
		}
	}
	return err
}

func leafXfmrHandlerFunc(inParams XfmrParams, xfmrFieldFuncNm string) (map[string]interface{}, error) {
	const (
		DBTY_FLD_XFMR_RET_ARGS     = 2
		DBTY_FLD_XFMR_RET_VAL_INDX = 0
		DBTY_FLD_XFMR_RET_ERR_INDX = 1
	)

	var err error
	var fldValMap map[string]interface{}

	xfmrLogDebug("Before calling dbToYang field xfmr %v, inParams %v", xfmrFieldFuncNm, inParams)
	ret, err := XlateFuncCall(dbToYangXfmrFunc(xfmrFieldFuncNm), inParams)
	xfmrLogDebug("After calling dbToYang field xfmr %v, inParams %v", xfmrFieldFuncNm, inParams)
	if err != nil {
		return fldValMap, err
	}
	if (ret != nil) && (len(ret) > 0) {
		if len(ret) == DBTY_FLD_XFMR_RET_ARGS {
			// field xfmr returns err as second value in return data list from <xfmr_func>.Call()
			if ret[DBTY_FLD_XFMR_RET_ERR_INDX].Interface() != nil {
				err = ret[DBTY_FLD_XFMR_RET_ERR_INDX].Interface().(error)
				if err != nil {
					log.Warningf("xfmr function(\"%v\") returned error - %v.", xfmrFieldFuncNm, err)
				}
			}
		}

		if ret[DBTY_FLD_XFMR_RET_VAL_INDX].Interface() != nil {
			fldValMap = ret[DBTY_FLD_XFMR_RET_VAL_INDX].Interface().(map[string]interface{})
			xfmrLogDebug("Field transformer returned %v", fldValMap)
		}
	}
	return fldValMap, err
}

func keyXfmrHandlerFunc(inParams XfmrParams, xfmrFuncNm string) (map[string]interface{}, error) {
	const (
		DBTY_KEY_XFMR_RET_ARGS     = 2
		DBTY_KEY_XFMR_RET_VAL_INDX = 0
		DBTY_KEY_XFMR_RET_ERR_INDX = 1
	)
	xfmrLogDebug("Before calling dbToYang key xfmr %v, inParams %v", xfmrFuncNm, inParams)
	ret, err := XlateFuncCall(dbToYangXfmrFunc(xfmrFuncNm), inParams)
	xfmrLogDebug("After calling dbToYang key xfmr %v, inParams %v", xfmrFuncNm, inParams)
	retVal := make(map[string]interface{})
	if err != nil {
		return retVal, err
	}
	if (ret != nil) && (len(ret) > 0) {
		if len(ret) == DBTY_KEY_XFMR_RET_ARGS {
			// key xfmr returns err as second value in return data list from <xfmr_func>.Call()
			if ret[DBTY_KEY_XFMR_RET_ERR_INDX].Interface() != nil {
				err = ret[DBTY_KEY_XFMR_RET_ERR_INDX].Interface().(error)
				if err != nil {
					log.Warningf("xfmr function(\"%v\") returned error - %v.", xfmrFuncNm, err)
					return retVal, err
				}
			}
		}
		if ret[DBTY_KEY_XFMR_RET_VAL_INDX].Interface() != nil {
			retVal = ret[DBTY_KEY_XFMR_RET_VAL_INDX].Interface().(map[string]interface{})
			xfmrLogDebug("Key transformer returned %v", retVal)
			return retVal, nil
		}
	}
	return retVal, nil
}

func validateHandlerFunc(inParams XfmrParams, validateFuncNm string) bool {
	xfmrLogDebug("Before calling validate xfmr %v, inParams %v", validateFuncNm, inParams)
	ret, err := XlateFuncCall(validateFuncNm, inParams)
	xfmrLogDebug("After calling validate xfmr %v, inParams %v", validateFuncNm, inParams)
	if err != nil {
		return false
	}
	result := ret[0].Interface().(bool)
	xfmrLogDebug("Validate transformer returned %v", result)
	return result
}

func xfmrTblHandlerFunc(xfmrTblFunc string, inParams XfmrParams, xfmrTblKeyCache map[string]tblKeyCache) ([]string, error) {
	const (
		TBL_XFMR_RET_ARGS     = 2
		TBL_XFMR_RET_VAL_INDX = 0
		TBL_XFMR_RET_ERR_INDX = 1
	)
	xfmrLogDebug("Before calling table xfmr %v, inParams %v", xfmrTblFunc, inParams)
	if inParams.oper == GET && xfmrTblKeyCache != nil {
		if tkCache, _ok := xfmrTblKeyCache[inParams.uri]; _ok {
			if len(tkCache.dbTblList) > 0 {
				xfmrLogDebug("Returning table list from cache %v", tkCache.dbTblList)
				return tkCache.dbTblList, nil
			}
		}
	}

	var retTblLst []string
	ret, err := XlateFuncCall(xfmrTblFunc, inParams)
	xfmrLogDebug("After calling table xfmr %v, inParams %v", xfmrTblFunc, inParams)
	if err != nil {
		return retTblLst, err
	}
	if (ret != nil) && (len(ret) > 0) {
		if len(ret) == TBL_XFMR_RET_ARGS {
			// table xfmr returns err as second value in return data list from <xfmr_func>.Call()
			if ret[TBL_XFMR_RET_ERR_INDX].Interface() != nil {
				err = ret[TBL_XFMR_RET_ERR_INDX].Interface().(error)
				if err != nil {
					log.Warningf("xfmr function(\"%v\") returned error - %v.", xfmrTblFunc, err)
					return retTblLst, err
				}
			}
		}

		if ret[TBL_XFMR_RET_VAL_INDX].Interface() != nil {
			retTblLst = ret[TBL_XFMR_RET_VAL_INDX].Interface().([]string)
		}
	}
	if inParams.oper == GET && xfmrTblKeyCache != nil {
		if _, _ok := xfmrTblKeyCache[inParams.uri]; !_ok {
			xfmrTblKeyCache[inParams.uri] = tblKeyCache{}
		}
		tkCache := xfmrTblKeyCache[inParams.uri]
		tkCache.dbTblList = retTblLst
		xfmrTblKeyCache[inParams.uri] = tkCache
	}

	xfmrLogDebug("Table transformer returned : %v", retTblLst)
	return retTblLst, err
}

func valueXfmrHandler(inParams XfmrDbParams, xfmrValueFuncNm string) (string, error) {
	const (
		YTDB_FLD_XFMR_RET_ARGS     = 2
		YTDB_FLD_XFMR_RET_VAL_INDX = 0
		YTDB_FLD_XFMR_RET_ERR_INDX = 1
	)

	xfmrLogDebug("Before calling value xfmr %v, inParams %v", xfmrValueFuncNm, inParams)
	ret, err := XlateFuncCall(xfmrValueFuncNm, inParams)
	xfmrLogDebug("After calling value xfmr %v, inParams %v", xfmrValueFuncNm, inParams)
	if err != nil {
		return "", err
	}

	if (ret != nil) && (len(ret) > 0) {
		if len(ret) == YTDB_FLD_XFMR_RET_ARGS {
			// value xfmr returns err as second value in return data list from <xfmr_func>.Call()
			if ret[YTDB_FLD_XFMR_RET_ERR_INDX].Interface() != nil {
				err = ret[YTDB_FLD_XFMR_RET_ERR_INDX].Interface().(error)
				if err != nil {
					log.Warningf("Transformer function(\"%v\") returned error - %v.", xfmrValueFuncNm, err)
					return "", err
				}
			}
		}

		if ret[YTDB_FLD_XFMR_RET_VAL_INDX].Interface() != nil {
			retVal := ret[YTDB_FLD_XFMR_RET_VAL_INDX].Interface().(string)
			xfmrLogDebug("Value transformer returned : %v", retVal)
			return retVal, nil
		}
	}

	return "", err
}

func leafXfmrHandler(inParams XfmrParams, xfmrFieldFuncNm string) (map[string]string, error) {
	const (
		YTDB_FLD_XFMR_RET_ARGS     = 2
		YTDB_FLD_XFMR_RET_VAL_INDX = 0
		YTDB_FLD_XFMR_RET_ERR_INDX = 1
	)

	xfmrLogDebug("Before calling yangToDb field xfmr %v, inParams %v", xfmrFieldFuncNm, inParams)
	ret, err := XlateFuncCall(yangToDbXfmrFunc(xfmrFieldFuncNm), inParams)
	xfmrLogDebug("After calling yangToDb field xfmr %v, inParams %v", xfmrFieldFuncNm, inParams)
	if err != nil {
		return nil, err
	}
	if (ret != nil) && (len(ret) > 0) {
		if len(ret) == YTDB_FLD_XFMR_RET_ARGS {
			// field xfmr returns err as second value in return data list from <xfmr_func>.Call()
			if ret[YTDB_FLD_XFMR_RET_ERR_INDX].Interface() != nil {
				err = ret[YTDB_FLD_XFMR_RET_ERR_INDX].Interface().(error)
				if err != nil {
					log.Warningf("xfmr function(\"%v\") returned error - %v.", xfmrFieldFuncNm, err)
					return nil, err
				}
			}
		}

		if ret[YTDB_FLD_XFMR_RET_VAL_INDX].Interface() != nil {
			fldValMap := ret[YTDB_FLD_XFMR_RET_VAL_INDX].Interface().(map[string]string)
			xfmrLogDebug("Field transformer returned %v", fldValMap)
			return fldValMap, nil
		}
	} else {
		retFldValMap := map[string]string{"NULL": "NULL"}
		return retFldValMap, nil
	}

	return nil, nil
}

func xfmrHandler(inParams XfmrParams, xfmrFuncNm string) (map[string]map[string]db.Value, error) {
	const (
		YTDB_SBT_XFMR_RET_ARGS     = 2
		YTDB_SBT_XFMR_RET_VAL_INDX = 0
		YTDB_SBT_XFMR_RET_ERR_INDX = 1
	)

	xfmrLogDebug("Before calling yangToDb subtree xfmr %v, inParams %v", xfmrFuncNm, inParams)
	ret, err := XlateFuncCall(yangToDbXfmrFunc(xfmrFuncNm), inParams)
	xfmrLogDebug("After calling yangToDb subtree xfmr %v, inParams %v", xfmrFuncNm, inParams)
	if err != nil {
		return nil, err
	}

	if (ret != nil) && (len(ret) > 0) {
		if len(ret) == YTDB_SBT_XFMR_RET_ARGS {
			// subtree xfmr returns err as second value in return data list from <xfmr_func>.Call()
			if ret[YTDB_SBT_XFMR_RET_ERR_INDX].Interface() != nil {
				err = ret[YTDB_SBT_XFMR_RET_ERR_INDX].Interface().(error)
				if err != nil {
					log.Warningf("xfmr function(\"%v\") returned error - %v.", xfmrFuncNm, err)
					return nil, err
				}
			}
		}
		if ret[YTDB_SBT_XFMR_RET_VAL_INDX].Interface() != nil {
			retMap := ret[YTDB_SBT_XFMR_RET_VAL_INDX].Interface().(map[string]map[string]db.Value)
			xfmrLogDebug("Subtree function returned %v", retMap)
			return retMap, nil
		}
	}
	return nil, nil
}

func keyXfmrHandler(inParams XfmrParams, xfmrFuncNm string) (string, error) {
	const (
		YTDB_KEY_XFMR_RET_ARGS     = 2
		YTDB_KEY_XFMR_RET_VAL_INDX = 0
		YTDB_KEY_XFMR_RET_ERR_INDX = 1
	)

	xfmrLogDebug("Before calling yangToDb key xfmr %v, inParams %v", xfmrFuncNm, inParams)
	ret, err := XlateFuncCall(yangToDbXfmrFunc(xfmrFuncNm), inParams)
	xfmrLogDebug("After calling yangToDb key xfmr %v, inParams %v", xfmrFuncNm, inParams)
	retVal := ""
	if err != nil {
		return retVal, err
	}

	if (ret != nil) && (len(ret) > 0) {
		if len(ret) == YTDB_KEY_XFMR_RET_ARGS {
			// key xfmr returns err as second value in return data list from <xfmr_func>.Call()
			if ret[YTDB_KEY_XFMR_RET_ERR_INDX].Interface() != nil {
				err = ret[YTDB_KEY_XFMR_RET_ERR_INDX].Interface().(error)
				if err != nil {
					log.Warningf("xfmr function(\"%v\") returned error - %v.", xfmrFuncNm, err)
					return retVal, err
				}
			}
		}
		if ret[YTDB_KEY_XFMR_RET_VAL_INDX].Interface() != nil {
			retVal = ret[YTDB_KEY_XFMR_RET_VAL_INDX].Interface().(string)
			xfmrLogDebug("Key xfmr returned %v", retVal)
			return retVal, nil
		}
	}
	return retVal, nil
}

/* Invoke the post tansformer */
func postXfmrHandlerFunc(xfmrPost string, inParams XfmrParams) error {
	const POST_XFMR_RET_ERR_INDX = 0
	xfmrLogDebug("Before calling post xfmr %v, inParams %v", xfmrPost, inParams)
	ret, err := XlateFuncCall(xfmrPost, inParams)
	xfmrLogDebug("After calling post xfmr %v, inParams %v", xfmrPost, inParams)
	if err != nil {
		return err
	}
	if (ret != nil) && (len(ret) > 0) {
		// post xfmr returns err as the only value in return data list from <xfmr_func>.Call()
		if ret[POST_XFMR_RET_ERR_INDX].Interface() != nil {
			err = ret[POST_XFMR_RET_ERR_INDX].Interface().(error)
			if err != nil {
				log.Warningf("Transformer function(\"%v\") returned error - %v.", xfmrPost, err)
				return err
			}
		}
	}
	return err
}

/* Invoke the pre tansformer */
func preXfmrHandlerFunc(xfmrPre string, inParams XfmrParams) error {
	const (
		PRE_XFMR_RET_ARGS     = 1
		PRE_XFMR_RET_ERR_INDX = 0
	)

	xfmrLogDebug("Before calling pre xfmr %v, inParams %v", xfmrPre, inParams)
	ret, err := XlateFuncCall(xfmrPre, inParams)
	xfmrLogDebug("After calling pre xfmr %v, inParams %v", xfmrPre, inParams)
	if err != nil {
		log.Warningf("Pre-transformer function(\"%v\") returned error - %v.", xfmrPre, err)
		return err
	}
	if (ret != nil) && (len(ret) > 0) {
		if ret[PRE_XFMR_RET_ERR_INDX].Interface() != nil {
			err = ret[PRE_XFMR_RET_ERR_INDX].Interface().(error)
			if err != nil {
				log.Warningf("Transformer function(\"%v\") returned error - %v.", xfmrPre, err)
				return err
			}
		}
	}
	return err
}

func sonicKeyXfmrHandlerFunc(inParams SonicXfmrParams, xfmrKeyNm string) (map[string]interface{}, error) {
	const (
		DBTY_KEY_XFMR_RET_ARGS     = 2
		DBTY_KEY_XFMR_RET_VAL_INDX = 0
		DBTY_KEY_XFMR_RET_ERR_INDX = 1
	)

	xfmrLogDebug("Before calling dbToYang sonic key xfmr %v, inParams %v", xfmrKeyNm, inParams)
	ret, err := XlateFuncCall(dbToYangXfmrFunc(xfmrKeyNm), inParams)
	xfmrLogDebug("After calling dbToYang sonic key xfmr %v, inParams %v", xfmrKeyNm, inParams)
	retVal := make(map[string]interface{})
	if err != nil {
		return retVal, err
	}

	if (ret != nil) && (len(ret) > 0) {
		if len(ret) == DBTY_KEY_XFMR_RET_ARGS {
			// key xfmr returns err as second value in return data list from <xfmr_func>.Call()
			if ret[DBTY_KEY_XFMR_RET_ERR_INDX].Interface() != nil {
				err = ret[DBTY_KEY_XFMR_RET_ERR_INDX].Interface().(error)
				if err != nil {
					log.Warningf("Transformer function(\"%v\") returned error - %v.", xfmrKeyNm, err)
					return retVal, err
				}
			}
		}
		if ret[DBTY_KEY_XFMR_RET_VAL_INDX].Interface() != nil {
			retVal = ret[DBTY_KEY_XFMR_RET_VAL_INDX].Interface().(map[string]interface{})
			xfmrLogDebug("Sonic key transformer returned %v", retVal)
			return retVal, nil
		}
	}
	return retVal, nil
}

func xfmrSubscSubtreeHandler(inParams XfmrSubscInParams, xfmrFuncNm string) (XfmrSubscOutParams, error) {
	const (
		SUBSC_SBT_XFMR_RET_ARGS     = 2
		SUBSC_SBT_XFMR_RET_VAL_INDX = 0
		SUBSC_SBT_XFMR_RET_ERR_INDX = 1
	)
	var retVal XfmrSubscOutParams
	retVal.dbDataMap = nil
	retVal.needCache = false
	retVal.onChange = OnchangeDisable
	retVal.nOpts = nil
	retVal.isVirtualTbl = false

	xfmrLogInfo("Received inParams %v Subscribe Subtree function name %v", inParams, xfmrFuncNm)
	ret, err := XlateFuncCall("Subscribe_"+xfmrFuncNm, inParams)
	if err != nil {
		return retVal, err
	}

	if (ret != nil) && (len(ret) > 0) {
		if len(ret) == SUBSC_SBT_XFMR_RET_ARGS {
			// subtree xfmr returns err as second value in return data list from <xfmr_func>.Call()
			if ret[SUBSC_SBT_XFMR_RET_ERR_INDX].Interface() != nil {
				err = ret[SUBSC_SBT_XFMR_RET_ERR_INDX].Interface().(error)
				if err != nil {
					log.Warningf("Subscribe xfmr function(\"%v\") returned error - %v.", xfmrFuncNm, err)
					return retVal, err
				}
			}
		}
		if ret[SUBSC_SBT_XFMR_RET_VAL_INDX].Interface() != nil {
			retVal = ret[SUBSC_SBT_XFMR_RET_VAL_INDX].Interface().(XfmrSubscOutParams)
		}
	}
	return retVal, err
}
