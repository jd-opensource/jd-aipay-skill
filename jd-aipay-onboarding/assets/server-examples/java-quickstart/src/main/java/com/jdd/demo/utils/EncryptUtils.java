package com.jdd.demo.utils;

import com.jdd.demo.common.Constants;
import com.wangyin.aks.pdf.util.PKCS12Keystore;
import com.wangyin.aks.security.api.CryptoClientService;
import com.wangyin.aks.security.api.CryptoClientServiceImpl;
import com.wangyin.aks.security.api.util.FileUtil;
import org.apache.commons.codec.binary.Base64;
import org.apache.commons.logging.Log;
import org.apache.commons.logging.LogFactory;

import java.util.List;

/**
 * 加密工具类
 */
public class EncryptUtils {
    private static Log logger = LogFactory.getLog(EncryptUtils.class);

    /**
     * 京东公钥
     **/
    private final static String SM2_JD_PUB = "MIICDDCCAbCgAwIBAgIUOlRimfnGphNGG+aMZ+oa3GANGRAwDAYIKoEcz1UBg3UFADBuMR4wHAYDVQQDDBVKUi5KRC5jb20gU00yIFRlc3QgQ0ExJDAiBgNVBAsMG1dhbmdZaW4gU2VjdXJpdHlDZW50ZXIgVEVTVDEZMBcGA1UECgwQV2FuZ1lpbi5jb20gVEVTVDELMAkGA1UEBhMCQ04wHhcNMjIwNzA3MDc0MTE3WhcNMjMwNzA3MDc0MTE3WjA+MRkwFwYDVQQDDBBQVEEoQUtTMDAwMDBBS1MpMRMwEQYDVQQLDApqciBzbTIgdG9wMQwwCgYDVQQKDANKREQwWTATBgcqhkjOPQIBBggqgRzPVQGCLQNCAATQYgxNkc+BzPJjRS2CWuusFYOcSCE9zrhRWUFs+6Or3dkQ9QK2s+kvnCq1NcLiJjFtZ0zdcS/jmco63dtMNRw1o1owWDAJBgNVHRMEAjAAMAsGA1UdDwQEAwIGwDAfBgNVHSMEGDAWgBS3NSmZBc0wTaWJp6b7QcNbfQ1W7jAdBgNVHQ4EFgQU519U1/dKQU82646KTzBdlRLGvaEwDAYIKoEcz1UBg3UFAANIADBFAiAdaX0Kh9mUGHKZH+MXQBnNURYATNge8zoowcBtc/6TGAIhAOBR97eqSxmsvshzS23FF2O0tPYhrTx6786odP7YsOxS";


    /**
     * SM2国密算法加密
     *
     * @param data   待加密数据
     * @param priUrl 私钥地址
     * @param passWord 私钥密码
     * @param pub    京东公钥
     * @return
     */
    public static String encryptForSm2(String data, String priUrl, String passWord, String pub) throws Exception {
        //数据转为字节数组
        byte[] dataBytes = data.getBytes(Constants.CHARSET_UTF8);
        //获取私钥
        CryptoClientService client = new CryptoClientServiceImpl();
        byte[] file = FileUtil.readFile(priUrl);
        PKCS12Keystore keystore = PKCS12Keystore.getInstance(file, passWord);
        //签名数字信封
        String signdigitalenvelope = client.signEnvelop(keystore, pub, dataBytes);
        return signdigitalenvelope;
    }

    /**
     * SM2国密算法加密（直接传入 pfx 字节内容）
     *
     * @param data     待加密数据
     * @param priBytes pfx 私钥文件内容（字节数组）
     * @param passWord 私钥密码
     * @param pub      京东公钥
     */
    public static String encryptForSm2WithBytes(String data, byte[] priBytes, String passWord, String pub) throws Exception {
        byte[] dataBytes = data.getBytes(Constants.CHARSET_UTF8);
        CryptoClientService client = new CryptoClientServiceImpl();
        PKCS12Keystore keystore = PKCS12Keystore.getInstance(priBytes, passWord);
        return client.signEnvelop(keystore, pub, dataBytes);
    }

    /**
     * SM2国密算法加密（直接传入 pfx 的 Base64 字符串）
     *
     * @param data      待加密数据
     * @param priBase64 pfx 私钥文件的 Base64 编码字符串
     * @param passWord  私钥密码
     * @param pub       京东公钥
     */
    public static String encryptForSm2WithBase64(String data, String priBase64, String passWord, String pub) throws Exception {
        byte[] priBytes = Base64.decodeBase64(priBase64);
        return encryptForSm2WithBytes(data, priBytes, passWord, pub);
    }

    /**
     * SM2国密算法解密（直接传入 pfx 的 Base64 字符串）
     *
     * @param encryptData SM2 数字信封密文
     * @param priBase64   pfx 私钥文件的 Base64 编码字符串
     * @param passWord    私钥密码
     */
    public static String decryptForSm2WithBase64(String encryptData, String priBase64, String passWord) throws Exception {
        byte[] priBytes = Base64.decodeBase64(priBase64);
        CryptoClientService client = new CryptoClientServiceImpl();
        PKCS12Keystore keystore = PKCS12Keystore.getInstance(priBytes, passWord);
        List<String> decryptEnvelop = client.verifyEnvelop(keystore, encryptData);
        String base64Plain = decryptEnvelop.get(0);
        byte[] byteData = new Base64().decode(base64Plain);
        return new String(byteData, Constants.CHARSET_UTF8);
    }

    /**
     * SM2国密算法解密
     * @param encryptData 待解密数据
     * @param priUrl 私钥地址
     * @param passWord 私钥密码
     * @return
     * @throws Exception
     */
    public static String decryptForSm2(String encryptData, String priUrl, String passWord) throws Exception {
        String decryData = null;
        //获取私钥
        CryptoClientService client = new CryptoClientServiceImpl();
        byte[] file = FileUtil.readFile(priUrl);
        PKCS12Keystore keystore = PKCS12Keystore.getInstance(file, passWord);
        //验证签名数字信封，并返回List[Base64格式原文，Base64格式签名者证书]
        List<String> decryptEnvelop = client.verifyEnvelop(keystore, encryptData);
        //获取Base64格式原文
        decryData = decryptEnvelop.get(0);
        //Base64格式转码为字符串
        Base64 base64 = new Base64();
        byte[] byteData = base64.decode(decryData);
        decryData = new String(byteData);
        System.out.println("decryData解密后数据= " + decryData);
        return decryData;
    }
}
